import Foundation
import Testing

@testable import Core

struct RuntimeManagerSupervisionTests {
    private static let restartDelays: [Duration] = [.seconds(1), .seconds(2), .seconds(4), .seconds(8), .seconds(16)]

    @Test func unexpectedSupervisorExitIsError() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.startRunning(0)

        handle.exit()

        #expect(await harness.reaches(.error))
        let message = await harness.manager.status(ManagerHarness.projectID).error ?? ""
        #expect(message.contains("runtime: supervisor exited unexpectedly"))
        #expect(await harness.persists(.error))
        #expect(harness.emitted().contains(.error))
    }

    @Test func readinessWatchdogStopsAStuckStart() async throws {
        let harness = try await ManagerHarness(stopIgnoredAt: [0])
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.launcher.handle(0)
        #expect(await harness.clock.waitForPending(RuntimeManager.readinessTimeout))

        harness.clock.advance(by: 60)

        #expect(await harness.reaches(.error))
        #expect(handle.stopCalls == 1)
        let watchdog = "watchdog killed runtime: never became ready in 1m0s"
        #expect(await Poll.until { try await harness.history().contains(watchdog) })
        #expect(await harness.manager.status(ManagerHarness.projectID).error == watchdog)
        handle.exit()
        #expect(await harness.persists(.error))
    }

    @Test func aReadyLineAfterTheStopOrderDoesNotPromoteTheSession() async throws {
        let harness = try await ManagerHarness(stopIgnoredAt: [0])
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.launcher.handle(0)
        let session = try #require(await harness.manager.sessions[ManagerHarness.projectID])

        session.requestShutdown()
        handle.write("  VITE ready in 300 ms")

        #expect(
            await Poll.until {
                await harness.manager.logs(ManagerHarness.projectID).contains { $0.text.contains("VITE ready") }
            })
        #expect(await harness.status() == .starting)
        #expect(!harness.emitted().contains(.running))
        handle.exit()
        #expect(await harness.persists(.stopped))
    }

    @Test func aStopDuringTheSelfHealWaitKeepsTheProjectDown() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        try await harness.startRunning(0).exit()
        #expect(await harness.clock.waitForPending(.seconds(1)))

        try? await harness.manager.stop(ManagerHarness.projectID)
        harness.clock.advance(by: 1)

        #expect(await !Poll.until(within: .milliseconds(300)) { harness.launcher.handles.count > 1 })
    }

    @Test func restartDoesNotLetTheOldSessionOverwriteStatus() async throws {
        let harness = try await ManagerHarness(stopIgnoredAt: [0])
        try await harness.manager.start(ManagerHarness.projectID)
        let old = try await harness.launcher.handle(0)
        #expect(await harness.clock.waitForPending(RuntimeManager.readinessTimeout))
        harness.clock.advance(by: 60)
        #expect(await harness.reaches(.error))
        harness.clock.advance(by: 11)

        try await harness.manager.start(ManagerHarness.projectID)
        let current = try await harness.startRunning(1)
        let emittedBefore = harness.emitted().count
        let oldSession = harness.launcher.specs[0].sessionID
        old.exit()
        #expect(await Poll.until { await harness.manager.supervising[oldSession] == nil })

        #expect(await harness.status() == .running)
        #expect(try await harness.persisted() == .running)
        #expect(harness.emitted().count == emittedBefore)
        current.exit()
    }

    @Test func crashAfterRunningSchedulesRestartWithBackoff() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)

        for attempt in 0...5 {
            let handle = try await harness.startRunning(attempt)
            handle.exit()
            guard attempt < Self.restartDelays.count else { break }
            let delay = Self.restartDelays[attempt]
            #expect(await harness.clock.waitForPending(delay))
            harness.clock.advance(by: TimeInterval(delay.components.seconds))
        }

        let givingUp = "self-heal giving up after 5 attempts"
        #expect(await Poll.until { try await harness.history().contains(givingUp) })
        #expect(await harness.reaches(.error))
        #expect(harness.emitted().last == .crash)
        #expect(harness.clock.requested.filter { $0 < RuntimeManager.readinessTimeout } == Self.restartDelays)
        #expect(harness.launcher.handles.count == 6)

        harness.clock.advance(by: 11)
        try await harness.manager.start(ManagerHarness.projectID)
        try await harness.startRunning(6).exit()
        #expect(await harness.clock.waitForPending(.seconds(1)))
    }

    @Test func crashBeforeRunningEmitsCrashWithoutRestart() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.launcher.handle(0)

        handle.exit()

        #expect(await harness.reaches(.error))
        #expect(await Poll.until { harness.emitted().last == .crash })
        #expect(await Poll.until { await harness.manager.supervising.isEmpty })
        #expect(await harness.manager.restarts.isEmpty)
        #expect(harness.launcher.handles.count == 1)
    }

    private func delaysAfterACrash(
        runningFor seconds: TimeInterval, expecting next: Duration
    ) async throws -> [Duration] {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        for attempt in 0...1 {
            try await harness.startRunning(attempt).exit()
            let delay = Self.restartDelays[attempt]
            #expect(await harness.clock.waitForPending(delay))
            harness.clock.advance(by: TimeInterval(delay.components.seconds))
        }
        let handle = try await harness.startRunning(2)

        harness.clock.advance(by: seconds)
        handle.exit()

        #expect(await Poll.until { await harness.manager.supervising.isEmpty })
        #expect(await harness.clock.waitForPending(next))
        return harness.clock.requested.filter { $0 < RuntimeManager.readinessTimeout }
    }

    @Test func crashAfterAStableRunWaitsOneSecondAgain() async throws {
        let delays = try await delaysAfterACrash(runningFor: 60, expecting: .seconds(1))

        #expect(delays == [.seconds(1), .seconds(2), .seconds(1)])
    }

    @Test func crashJustBeforeTheStabilityWindowKeepsCounting() async throws {
        let delays = try await delaysAfterACrash(runningFor: 59, expecting: .seconds(4))

        #expect(delays == [.seconds(1), .seconds(2), .seconds(4)])
    }

    @Test func aOneMebibyteLineIsCutInMemoryAndInTheStoreAndStillDetectsReadiness() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.launcher.handle(0)
        let filler = String(repeating: "x", count: (1 << 20) - 100)

        handle.write("\(filler) VITE ready in 10 ms")

        #expect(await harness.reaches(.running))
        let line = try #require(await harness.manager.logs(ManagerHarness.projectID).last)
        #expect(line.text.utf8.count == StoredText.maximumBytes)
        let stored = try await harness.manager.history(ManagerHarness.projectID, since: 0, limit: 0)
        #expect(stored.contains { $0.text == line.text })
        handle.exit()
    }
}
