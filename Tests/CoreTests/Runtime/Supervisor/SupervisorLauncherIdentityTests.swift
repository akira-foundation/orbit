import Darwin
import Testing

@testable import Core

struct SupervisorLauncherIdentityTests {
    private static func launch(_ inspector: ScriptedLeaderInspector) async throws -> any SupervisorHandle {
        try await SupervisorLauncher(executable: ProcessFixture.runtimeFixturePath, inspector: inspector).launch(
            SupervisedRuntime.spec(["/bin/sleep", "600"], grace: .milliseconds(20)))
    }

    @Test func launchWaitsForTheSupervisorToLeadItsGroup() async throws {
        let handle = try await Self.launch(ScriptedLeaderInspector([.parentGroup, .parentGroup, .leader]))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        #expect(handle.identity.pgid == handle.identity.pid)
        try handle.stop()
        try await handle.wait()
    }

    @Test func launchRetriesAnIdentityThatChangedDuringEnrichment() async throws {
        let handle = try await Self.launch(ScriptedLeaderInspector([.changed, .changed, .leader]))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        try handle.stop()
        try await handle.wait()
    }

    @Test func verifyEmptyFailsWhileAGroupMemberIsUnsettled() async throws {
        let inspector = ScriptedLeaderInspector([.leader], group: ProcessListing(identities: [], unsettled: [4242]))
        let handle = try await Self.launch(inspector)
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        let error = await #expect(throws: RuntimeErrors.self) {
            try await handle.verifyEmpty(within: .milliseconds(20))
        }

        #expect(error?.contains(.groupNotEmpty) == true)
        try handle.stop()
        try await handle.wait()
    }

    @Test func launchGivesUpOnASupervisorThatNeverLeadsItsGroup() async throws {
        let inspector = ScriptedLeaderInspector([.parentGroup])
        defer { inspector.inspectedPIDs.forEach(SupervisedRuntime.killGroup) }
        let clock = ContinuousClock()
        let started = clock.now

        let error = await #expect(throws: RuntimeError.self) { try await Self.launch(inspector) }

        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(clock.now - started >= .seconds(1))
    }

    @Test func cancellingALaunchThatIsWaitingForTheLeaderReportsCancellation() async throws {
        let inspector = ScriptedLeaderInspector([.parentGroup])
        defer { inspector.inspectedPIDs.forEach(SupervisedRuntime.killGroup) }
        let launching = Task { try await Self.launch(inspector) }
        #expect(await Poll.until { !inspector.inspectedPIDs.isEmpty })
        launching.cancel()

        await #expect(throws: RuntimeError.cancelled) { try await launching.value }
    }
}
