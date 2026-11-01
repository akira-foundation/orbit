import Darwin
import Testing

@testable import Core

struct SupervisorLauncherIdentityTests {
    private static func launch(_ reads: [ScriptedLeaderInspector.Read]) async throws -> any SupervisorHandle {
        try await SupervisorLauncher(
            executable: ProcessFixture.runtimeFixturePath, inspector: ScriptedLeaderInspector(reads)
        ).launch(SupervisedRuntime.spec(["/bin/sleep", "600"], grace: .milliseconds(20)))
    }

    @Test func launchWaitsForTheSupervisorToLeadItsGroup() async throws {
        let handle = try await Self.launch([.parentGroup, .parentGroup, .leader])
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        #expect(handle.identity.pgid == handle.identity.pid)
        try handle.stop()
        try await handle.wait()
    }

    @Test func launchRetriesAnIdentityThatChangedDuringEnrichment() async throws {
        let handle = try await Self.launch([.changed, .changed, .leader])
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        try handle.stop()
        try await handle.wait()
    }

    @Test func launchGivesUpOnASupervisorThatNeverLeadsItsGroup() async throws {
        let clock = ContinuousClock()
        let started = clock.now

        let error = await #expect(throws: RuntimeError.self) { try await Self.launch([.parentGroup]) }

        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(clock.now - started >= .seconds(1))
    }

    @Test func cancellingALaunchThatIsWaitingForTheLeaderReportsCancellation() async throws {
        let launching = Task { try await Self.launch([.parentGroup]) }
        try await Task.sleep(for: .milliseconds(100))
        launching.cancel()

        await #expect(throws: RuntimeError.cancelled) { try await launching.value }
    }
}
