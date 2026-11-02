import Darwin
import Foundation
import Testing

@testable import Core

struct SupervisorShutdownTests {
    private static let execOnTerm = "trap 'exec /bin/sleep 600' TERM; echo READY; /bin/sleep 600 & wait"

    @Test func childThatExecsOnTermIsKilledAndTheSupervisorExitsCleanly() async throws {
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(["/bin/sh", "-c", Self.execOnTerm], grace: .milliseconds(20)))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }
        _ = try SupervisedRuntime.readLine(handle, prefix: "READY")

        try handle.stop()
        let outcome = await SupervisedRuntime.waitBounded(handle, within: .seconds(20))

        #expect(throws: Never.self) { try #require(outcome).get() }
        #expect(await SupervisedRuntime.groupIsEmpty(handle.identity.pgid, within: .seconds(20)))
    }

    @Test func failedCleanupStillKillsTheDirectChildAndExits() async throws {
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec([ProcessFixture.runtimeFixturePath, "--setsid-on-term"], grace: .milliseconds(20)))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }
        _ = try SupervisedRuntime.readLine(handle, prefix: "READY")

        try handle.stop()
        let outcome = await SupervisedRuntime.waitBounded(handle, within: .seconds(10))

        #expect(throws: RuntimeError.message("supervisor exit status 1")) { try #require(outcome).get() }
        #expect(await SupervisedRuntime.groupIsEmpty(handle.identity.pgid, within: .seconds(5)))
    }

    @Test func droppingTheHandleWithoutStopCleansTheGroup() async throws {
        let (pgid, grandchildPID, grandchild) = try await {
            let runtime = try await SupervisedRuntime.start(SupervisedRuntime.holds, grace: .milliseconds(20))
            return (runtime.handle.identity.pgid, runtime.grandchildPID, runtime.grandchild)
        }()
        defer {
            SupervisedRuntime.killGroup(pgid)
            SupervisedRuntime.killVerified(grandchild)
        }

        #expect(grandchild != nil)
        #expect(await SupervisedRuntime.groupIsEmpty(pgid, within: .seconds(5)))
        #expect(await SupervisedRuntime.isGone(grandchildPID, was: grandchild, within: .seconds(5)))
    }

    @Test func errorLineWithAClosedStderrDoesNotCrashTheSupervisor() async throws {
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(["/bin/sh", "-c", "/bin/sleep 0.5; exit 3"], grace: .milliseconds(20)))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        try handle.stderr?.close()

        await #expect(throws: RuntimeError.message("supervisor exit status 1")) { try await handle.wait() }
    }

    @Test func relativeCommandResolvesAgainstTheWorkingDirectory() async throws {
        let directory = try TemporaryDirectory()
        try directory.write("#!/bin/sh\n/bin/sleep 0.5\necho RAN_IN_$(/bin/pwd -P)\n", to: "run.sh")
        chmod(directory.path + "/run.sh", 0o755)
        var spec = SupervisedRuntime.spec(["./run.sh"], grace: .milliseconds(20))
        spec.directory = directory.path
        let handle = try await SupervisedRuntime.launch(spec)
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        try await handle.wait()

        let output = try handle.stdout?.readToEnd().map { String(decoding: $0, as: UTF8.self) } ?? ""
        #expect(output.hasPrefix("RAN_IN_/") && output.hasSuffix(directory.path + "\n"))
    }

    @Test func bareCommandNameIsNotSearchedInPath() async throws {
        let handle: any SupervisorHandle
        do {
            handle = try await SupervisedRuntime.launch(
                SupervisedRuntime.spec(["sleep", "600"], grace: .milliseconds(20)))
        } catch {
            #expect("\(error)" == "runtime: inspect started supervisor: process already finished")
            return
        }
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        await #expect(throws: RuntimeError.message("supervisor exit status 1")) { try await handle.wait() }
        let stderr = try handle.stderr?.readToEnd().map { String(decoding: $0, as: UTF8.self) }
        #expect(stderr?.contains("No such file or directory") == true)
    }
}
