import Testing

@testable import Core

struct RuntimeManagerAbandonedLaunchTests {
    private func expectStoppedWithNothingAlive(_ harness: ManagerHarness) async throws {
        let handle = try await harness.launcher.handle(0)
        #expect(handle.stopCalls == 1)
        #expect(await harness.reaches(.stopped))
        #expect(await harness.persists(.stopped))
        #expect(await Poll.until { await harness.manager.supervising.isEmpty })
        #expect(await !harness.manager.isRunning(ManagerHarness.projectID))
        #expect(!harness.emitted().contains(.starting))
        #expect(harness.emitted().last == .stopped)
    }

    @Test func launchThatEndsAfterStopGaveUpIsStopped() async throws {
        let harness = try await ManagerHarness(initialStatus: .error)
        harness.launcher.holdLaunches()
        let start = Task { try await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.launcher.heldLaunches == 1 })

        try await harness.manager.stop(ManagerHarness.projectID)
        let stoppedWhileHeld = harness.services.stopped
        harness.launcher.releaseLaunches()
        try await start.value

        #expect(stoppedWhileHeld.isEmpty)
        try await expectStoppedWithNothingAlive(harness)
        #expect(harness.services.stopped == [ManagerHarness.projectID])
    }

    @Test func launchThatEndsAfterCloseGaveUpIsStopped() async throws {
        let harness = try await ManagerHarness(initialStatus: .error)
        harness.launcher.holdLaunches()
        let start = Task { try await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.launcher.heldLaunches == 1 })

        let close = Task { await harness.manager.close() }
        #expect(await harness.clock.waitForPending(RuntimeManager.stopAllTimeout))
        harness.clock.advance(by: 3)
        await close.value
        harness.launcher.releaseLaunches()
        try await start.value

        try await expectStoppedWithNothingAlive(harness)
        #expect(harness.services.stopped == [ManagerHarness.projectID])
    }
}
