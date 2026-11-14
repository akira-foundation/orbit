import Testing

@testable import Core

struct RuntimeManagerAttemptsTests {
    private func snapshots(_ harness: ManagerHarness, named name: RuntimeEventName) -> [SessionSnapshot] {
        harness.sink.events.compactMap { event in
            guard case .status(name, let snapshot) = event else { return nil }
            return snapshot
        }
    }

    @Test func statusAndEventsReportTheSelfHealAttempts() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let first = try await harness.startRunning(0)
        #expect(await harness.manager.status(ManagerHarness.projectID).attempts ?? 0 == 0)

        first.exit()

        #expect(await Poll.until { await harness.manager.status(ManagerHarness.projectID).attempts == 1 })
        #expect(await harness.clock.waitForPending(.seconds(1)))
        harness.clock.advance(by: 1)
        let second = try await harness.startRunning(1)
        #expect(await harness.manager.status(ManagerHarness.projectID).attempts == 1)
        #expect(snapshots(harness, named: .starting).last?.attempts == 1)
        #expect(snapshots(harness, named: .running).last?.attempts == 1)

        second.exit()

        #expect(await Poll.until { await harness.manager.status(ManagerHarness.projectID).attempts == 2 })
    }

    @Test func stoppingForgetsTheAttempts() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        try await harness.startRunning(0).exit()
        #expect(await Poll.until { await harness.manager.status(ManagerHarness.projectID).attempts == 1 })

        try? await harness.manager.stop(ManagerHarness.projectID)

        #expect(await harness.manager.status(ManagerHarness.projectID).attempts ?? 0 == 0)
    }
}
