import Darwin
import Testing

@testable import Core

struct SupervisorIsolationTests {
    private static let insertedLibrary = "/usr/lib/libSystem.B.dylib"

    @Test func supervisorLeadsASessionOfItsOwn() async throws {
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(["/bin/sleep", "600"], grace: .milliseconds(20)))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        #expect(getsid(handle.identity.pid) == handle.identity.pid)
        #expect(handle.identity.pgid == handle.identity.pid)
        #expect(getsid(0) != handle.identity.pid)
        try handle.stop()
        try await handle.wait()
    }

    @Test func aProjectCannotJoinTheGroupOfAnotherProject() async throws {
        let victim = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(["/bin/sleep", "600"], grace: .milliseconds(20)))
        defer { SupervisedRuntime.killGroup(victim.identity.pgid) }
        let intruder = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(
                [ProcessFixture.runtimeFixturePath, "--join-group", "\(victim.identity.pgid)"],
                grace: .milliseconds(20)))
        defer { SupervisedRuntime.killGroup(intruder.identity.pgid) }

        #expect(try SupervisedRuntime.readLine(intruder, prefix: "JOIN_ERRNO=") == "JOIN_ERRNO=\(EPERM)")
        try intruder.stop()
        try victim.stop()
        try await intruder.wait()
        try await victim.wait()
    }

    @Test func aChildJoinsTheGroupOfItsOwnSupervisor() async throws {
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(["/bin/sh", "-c", SupervisedRuntime.script(SupervisedRuntime.holds)], grace: .zero))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        try await SupervisedRuntime.waitForSleepingMembers(of: handle.identity)
        let members = try DarwinProcessInspector().groupMembers(handle.identity.pgid)
        #expect(members.map(\.pid).allSatisfy { getsid($0) == handle.identity.pid })
        try handle.stop()
    }

    @Test func dynamicLoaderVariablesFromTheEnvFileReachTheChildButNotTheSupervisor() async throws {
        let directory = try TemporaryDirectory()
        try directory.write("DYLD_INSERT_LIBRARIES=\(Self.insertedLibrary)\nAPP_NAME=orbit\n", to: ".env")
        var environment = EnvironmentList([])
        environment.override(DotEnv.load(directory: directory.path))
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(
                [ProcessFixture.runtimeFixturePath, "--print-environment", "DYLD_INSERT_LIBRARIES"],
                grace: .milliseconds(20), environment: environment))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        let line = try SupervisedRuntime.readLine(handle, prefix: "DYLD_INSERT_LIBRARIES=")
        #expect(line == "DYLD_INSERT_LIBRARIES=\(Self.insertedLibrary)")
        let supervisorStrings = try ProcessEnvironmentReader.strings(of: handle.identity.pid)
        #expect(supervisorStrings.contains("APP_NAME=orbit"))
        #expect(!supervisorStrings.contains { $0.hasPrefix("DYLD_INSERT_LIBRARIES=") })
        try handle.stop()
        try await handle.wait()
    }

    @Test func thePrefixCannotSmuggleOtherVariablesIntoTheChild() {
        var environment = EnvironmentList(["DYLD_LIBRARY_PATH=/x", "PATH=/bin", "ORBIT_RUNTIME_CHILD_PATH=/evil"])
        environment = SupervisorEnvironment.forSupervisor(environment)
        #expect(environment.value(of: "DYLD_LIBRARY_PATH") == nil)
        let child = SupervisorEnvironment.forChild(environment)
        #expect(child.value(of: "DYLD_LIBRARY_PATH") == "/x")
        #expect(child.value(of: "PATH") == "/bin")
        #expect(!child.entries.contains { $0.hasPrefix(SupervisorEnvironment.deferredLoaderPrefix) })
    }
}
