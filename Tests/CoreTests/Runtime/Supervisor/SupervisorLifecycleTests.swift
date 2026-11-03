import Darwin
import Testing

@testable import Core

struct SupervisorLifecycleTests {
    @Test func controlEOFKillsTermResistantGrandchild() async throws {
        let runtime = try await SupervisedRuntime.start(SupervisedRuntime.holds, grace: .milliseconds(20))
        defer { runtime.cleanUp() }
        #expect(runtime.grandchild != nil)

        try runtime.handle.stop()
        try await runtime.handle.wait()

        #expect(await runtime.grandchildGone())
    }

    @Test func runnerExitKillsGrandchild() async throws {
        let runtime = try await SupervisedRuntime.start(SupervisedRuntime.runnerExits, grace: .milliseconds(20))
        defer { runtime.cleanUp() }

        #expect(runtime.grandchild != nil)
        try await runtime.handle.wait()

        #expect(await runtime.grandchildGone())
    }

    @Test func concurrentStopCleansGroup() async throws {
        let runtime = try await SupervisedRuntime.start(SupervisedRuntime.holds, grace: .milliseconds(20))
        defer { runtime.cleanUp() }
        let handle = runtime.handle

        let failures = await withTaskGroup(of: Bool.self) { group in
            for _ in 0..<8 {
                group.addTask { (try? handle.stop()) == nil }
            }
            return await group.reduce(0) { $0 + ($1 ? 1 : 0) }
        }
        try await handle.wait()

        #expect(failures == 0)
        #expect(await runtime.grandchildGone())
    }

    @Test func signalTriggerCleansGroup() async throws {
        let runtime = try await SupervisedRuntime.start(SupervisedRuntime.holds, grace: .milliseconds(20))
        defer { runtime.cleanUp() }

        try DarwinProcessInspector().signal(runtime.handle.identity, .terminate)
        try await runtime.handle.wait()

        #expect(await runtime.grandchildGone())
    }

    @Test func interruptTriggerCleansGroup() async throws {
        let runtime = try await SupervisedRuntime.start(SupervisedRuntime.holds, grace: .milliseconds(20))
        defer { runtime.cleanUp() }

        try DarwinProcessInspector().signal(runtime.handle.identity, .custom(SIGINT))
        try await runtime.handle.wait()

        #expect(await runtime.grandchildGone())
    }

    @Test func runtimeThatKeepsDefaultTermDiesWithoutWaitingForTheGrace() async throws {
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(["/bin/sleep", "600"], grace: .seconds(30)))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }
        try await SupervisedRuntime.waitForMember("/bin/sleep 600", in: handle.identity.pgid)
        let clock = ContinuousClock()
        let started = clock.now

        try handle.stop()
        try await handle.wait()

        #expect(clock.now - started < .seconds(25))
    }

    @Test func waitSupportsCancellation() async throws {
        let runtime = try await SupervisedRuntime.start(SupervisedRuntime.holds, grace: .milliseconds(20))
        defer { runtime.cleanUp() }
        let handle = runtime.handle

        let waiting = Task { try await handle.wait() }
        waiting.cancel()

        await #expect(throws: RuntimeError.cancelled) { try await waiting.value }
        try handle.stop()
        try await handle.wait()
        #expect(await runtime.grandchildGone())
    }

    @Test func verifyEmptyConfirmsExactGroupEmptiness() async throws {
        let runtime = try await SupervisedRuntime.start(SupervisedRuntime.holds, grace: .milliseconds(20))
        defer { runtime.cleanUp() }
        let handle = runtime.handle

        let error = await #expect(throws: RuntimeErrors.self) {
            try await handle.verifyEmpty(within: .milliseconds(20))
        }
        #expect(error?.contains(.groupNotEmpty) == true && error?.contains(.cancelled) == true)

        try handle.stop()
        try await handle.wait()
        try await handle.verifyEmpty(within: .seconds(1))
    }

    @Test func runnerFailureFailsTheSupervisor() async throws {
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(["/bin/sh", "-c", "/bin/sleep 0.5; exit 3"], grace: .milliseconds(20)))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        await #expect(throws: RuntimeError.message("supervisor exit status 1")) { try await handle.wait() }
        let stderr = try handle.stderr?.readToEnd().map { String(decoding: $0, as: UTF8.self) }
        #expect(stderr == "exit status 3\n")
    }

    @Test func supervisorPassesItsEnvironmentWithTheGrace() async throws {
        let environment = EnvironmentList(["A=b", "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=1"])
        let handle = try await SupervisedRuntime.launch(
            SupervisedRuntime.spec(
                ["/bin/sh", "-c", "/bin/sleep 0.5; exec /usr/bin/env"], grace: .milliseconds(20),
                environment: environment))
        defer { SupervisedRuntime.killGroup(handle.identity.pgid) }

        try await handle.wait()

        let output = try handle.stdout?.readToEnd().map { String(decoding: $0, as: UTF8.self) } ?? ""
        let lines = output.split(separator: "\n").map(String.init)
        #expect(lines.contains("A=b"))
        #expect(
            lines.filter { $0.hasPrefix("ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=") } == [
                "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=20"
            ])
    }

    @Test func launchRejectsAnInvalidSpecBeforeSpawning() async {
        var spec = SupervisedRuntime.spec(["/bin/sleep", "600"], grace: .milliseconds(20))
        spec.directory = "relative"
        await #expect(throws: RuntimeError.message("runtime: supervisor cwd must be absolute")) {
            try await SupervisedRuntime.launch(spec)
        }
    }

    @Test func launchFailsWhenTheExecutableIsMissing() async {
        let launcher = SupervisorLauncher(executable: "/nonexistent/orbit", inspector: DarwinProcessInspector())
        await #expect(throws: RuntimeError.message("runtime: start supervisor: start: No such file or directory")) {
            try await launcher.launch(SupervisedRuntime.spec(["/bin/sleep", "600"], grace: .milliseconds(20)))
        }
    }
}
