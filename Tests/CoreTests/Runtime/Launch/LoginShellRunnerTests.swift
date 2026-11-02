import Foundation
import Synchronization
import Testing

@testable import Core

struct LoginShellRunnerTests {
    @Test func returnsStandardOutput() async {
        let output = await LoginShellRunner().run(shell: "/bin/sh", script: "echo hi", timeout: .seconds(5))
        #expect(output == "hi\n")
    }

    @Test func returnsNilAndStopsWaitingAfterTheTimeout() async {
        let started = ContinuousClock.now
        let output = await LoginShellRunner().run(shell: "/bin/sh", script: "sleep 30", timeout: .milliseconds(100))
        #expect(output == nil)
        #expect(ContinuousClock.now - started < .seconds(15))
    }

    @Test func returnsNilWhenTheShellDoesNotExist() async {
        let output = await LoginShellRunner().run(shell: "/nonexistent/sh", script: "echo hi", timeout: .seconds(1))
        #expect(output == nil)
    }

    @Test func aNonZeroExitReturnsNil() async {
        let output = await LoginShellRunner().run(shell: "/bin/sh", script: "echo hi; exit 3", timeout: .seconds(5))
        #expect(output == nil)
    }

    @Test func aShellKilledBySignalReturnsNil() async {
        let output = await LoginShellRunner().run(shell: "/bin/sh", script: "kill -9 $$", timeout: .seconds(5))
        #expect(output == nil)
    }

    @Test func largeOutputIsReadInFull() async {
        let output = await LoginShellRunner().run(
            shell: "/bin/sh", script: "head -c 1048576 /dev/zero | tr '\\0' a", timeout: .seconds(10))
        #expect(output?.utf8.count == 1_048_576)
    }

    @Test func outputAboveTheLimitReturnsNil() async {
        let output = await LoginShellRunner().run(
            shell: "/bin/sh", script: "head -c 8388608 /dev/zero | tr '\\0' a", timeout: .seconds(10))
        #expect(output == nil)
    }

    @Test func theTimeoutKillsTheWholeProcessGroup() async throws {
        let directory = try TemporaryDirectory()
        let pidFile = directory.path + "/grandchild.pid"
        let started = ContinuousClock.now
        let output = await LoginShellRunner().run(
            shell: "/bin/sh", script: "sleep 30 & echo $! > '\(pidFile)'; wait", timeout: .milliseconds(1500))
        let elapsed = ContinuousClock.now - started
        let pid = try #require(
            pid_t(String(contentsOfFile: pidFile, encoding: .utf8).trimmingCharacters(in: .whitespacesAndNewlines)))
        defer { kill(pid, SIGKILL) }
        #expect(output == nil)
        #expect(elapsed < .seconds(4))
        #expect(await processDisappears(pid))
    }

    @Test func aShellThatClosesStandardOutputIsPolledAtTheSleepIntervalNotInATightLoop() async {
        let iterations = Mutex(0)
        let runner = LoginShellRunner(onPollIteration: { iterations.withLock { $0 += 1 } })
        let output = await runner.run(shell: "/bin/sh", script: "exec >&-; sleep 1", timeout: .seconds(5))
        #expect(output == "")
        #expect(iterations.withLock { $0 } < 300)
    }

    @Test(.disabled(if: ThreadSanitizer.isLoaded, ThreadSanitizer.exitTestsUnsupported))
    func signalIgnoredByTheParentIsDefaultInTheShell() async {
        await #expect(processExitsWith: .success) {
            signal(SIGTERM, SIG_IGN)
            let output = await LoginShellRunner().run(
                shell: "/bin/sh", script: "/bin/sh -c 'kill -TERM $$; echo survived'", timeout: .seconds(5))
            exit(output?.contains("survived") == true ? EXIT_FAILURE : EXIT_SUCCESS)
        }
    }

    private func processDisappears(_ pid: pid_t) async -> Bool {
        for _ in 0..<100 {
            if kill(pid, 0) == -1 && errno == ESRCH {
                return true
            }
            try? await Task.sleep(for: .milliseconds(20))
        }
        return false
    }
}
