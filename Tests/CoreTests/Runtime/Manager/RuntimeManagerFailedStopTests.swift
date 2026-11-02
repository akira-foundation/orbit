import Testing

@testable import Core

struct RuntimeManagerFailedStopTests {
    private func expectStopFails(
        with failure: RuntimeError, _ configure: (ScriptedHandle) -> Void
    ) async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.startRunning(0)
        configure(handle)

        await #expect(throws: failure) { try await harness.manager.stop(ManagerHarness.projectID) }

        #expect(await harness.status() == .error)
        #expect(try await harness.persisted() == .error)
        #expect(harness.emitted().contains(.error))
        #expect(harness.services.stopped == [ManagerHarness.projectID])
        #expect(await Poll.until { await harness.manager.supervising.isEmpty })
        #expect(await harness.manager.status(ManagerHarness.projectID).attempts ?? 0 == 0)
        #expect(!harness.clock.requested.contains(.seconds(1)))
    }

    @Test func aStopThatLeavesTheGroupBehindThrowsAndPersistsTheError() async throws {
        try await expectStopFails(with: .groupNotEmpty) { $0.verifyFailure = .groupNotEmpty }
    }

    @Test func aStopWhoseSupervisorFailedThrowsAndPersistsTheError() async throws {
        let failure = RuntimeError.message("supervisor: exit status 3")
        try await expectStopFails(with: failure) { $0.waitFailure = failure }
    }

    @Test func anUnrequestedExitWithAFailedVerificationReportsBothCauses() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.startRunning(0)
        handle.verifyFailure = .groupNotEmpty

        handle.exit()

        #expect(await harness.reaches(.error))
        let message = await harness.manager.status(ManagerHarness.projectID).error ?? ""
        #expect(message.contains("\(RuntimeError.unexpectedSupervisorExit)"))
        #expect(message.contains("\(RuntimeError.groupNotEmpty)"))
        #expect(await harness.persists(.error))
    }
}
