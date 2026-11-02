import Testing

@testable import Core

struct RuntimeManagerOldSessionTests {
    @Test func aLaunchAbandonedByStopCannotOverwriteTheNewSessionStatus() async throws {
        let harness = try await ManagerHarness()
        harness.launcher.holdLaunches()
        let abandoned = Task { try? await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.launcher.heldLaunches == 1 })
        try await harness.manager.stop(ManagerHarness.projectID)
        let session = try #require(await harness.manager.sessions[ManagerHarness.projectID])
        session.markStopped()
        harness.launcher.stopHolding()
        try await harness.manager.start(ManagerHarness.projectID)
        let current = try await harness.startRunning(0)
        #expect(await harness.persists(.running))

        harness.launcher.failure = "boom"
        harness.launcher.releaseLaunches()
        await abandoned.value

        #expect(await harness.status() == .running)
        #expect(try await harness.persisted() == .running)
        current.exit()
    }

    @Test func outputOfTheOldSessionIsNotEmitted() async throws {
        let harness = try await ManagerHarness(stopIgnoredAt: [0])
        try await harness.manager.start(ManagerHarness.projectID)
        let old = try await harness.launcher.handle(0)
        let oldSession = try #require(await harness.manager.sessions[ManagerHarness.projectID])
        #expect(await harness.clock.waitForPending(RuntimeManager.readinessTimeout))
        harness.clock.advance(by: 60)
        #expect(await harness.reaches(.error))
        harness.clock.advance(by: 11)
        try await harness.manager.start(ManagerHarness.projectID)
        let current = try await harness.startRunning(1)

        old.write("old output")

        #expect(await Poll.until { oldSession.logs().contains { $0.text == "old output" } })
        let emitted = harness.sink.events.contains { event in
            guard case .log(_, let line) = event else { return false }
            return line.text == "old output"
        }
        #expect(!emitted)
        old.exit()
        current.exit()
    }
}
