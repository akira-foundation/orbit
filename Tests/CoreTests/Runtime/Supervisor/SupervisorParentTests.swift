import Darwin
import Testing

@testable import Core

struct SupervisorParentTests {
    @Test func parentDisappearanceCleansGroup() async throws {
        let launcher = try PosixSpawner.spawn(
            SpawnRequest(
                executable: ProcessFixture.runtimeFixturePath,
                arguments: ["--launch-supervisor", "/bin/sh", "-c", SupervisedRuntime.script(SupervisedRuntime.holds)],
                environment: EnvironmentList([]), captureOutput: true))
        let fixture = SpawnedTestProcess(adopting: launcher)
        defer { fixture.terminate() }

        let lines = try fixture.outputLines()
        let supervisorPID = try #require(Self.value("SUPERVISOR_PID=", in: lines))
        defer { SupervisedRuntime.killGroup(supervisorPID) }
        let grandchildPID = try #require(Self.value("GRANDCHILD_PID=", in: lines))
        let grandchild = await SupervisedRuntime.settledSleeper(grandchildPID, waitForIt: false)
        defer { SupervisedRuntime.killVerified(grandchild) }

        #expect(try await launcher.exitStatus() == .exited(0))
        #expect(await SupervisedRuntime.groupIsEmpty(supervisorPID, within: .seconds(5)))
        #expect(await SupervisedRuntime.isGone(grandchildPID, was: grandchild, within: .seconds(5)))
    }

    private static func value(_ prefix: String, in lines: [String]) -> Int32? {
        lines.first { $0.hasPrefix(prefix) }.flatMap { Int32($0.dropFirst(prefix.count)) }
    }
}
