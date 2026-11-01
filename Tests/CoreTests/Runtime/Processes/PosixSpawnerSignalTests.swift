import Darwin
import Testing

@testable import Core

struct PosixSpawnerSignalTests {
    @Test(.disabled(if: ThreadSanitizer.isLoaded, ThreadSanitizer.exitTestsUnsupported))
    func signalIgnoredByTheParentIsDefaultInTheChild() async {
        await #expect(processExitsWith: .success) {
            signal(SIGTERM, SIG_IGN)
            let process = try PosixSpawner.spawn(
                SpawnRequest(executable: "/bin/sleep", arguments: ["30"], environment: EnvironmentList([])))
            kill(process.pid, SIGTERM)
            let status = try? await ReapedStatus.await(process)
            if status == nil {
                kill(-process.pid, SIGKILL)
            }
            exit(status == .signaled(SIGTERM) ? EXIT_SUCCESS : EXIT_FAILURE)
        }
    }

    @Test func signalBlockedByTheParentThreadIsUnblockedInTheChild() async throws {
        var blocked: sigset_t = 1 << (SIGTERM - 1)
        var previous = sigset_t()
        pthread_sigmask(SIG_BLOCK, &blocked, &previous)
        let spawned = Result { try sleeper() }
        pthread_sigmask(SIG_SETMASK, &previous, nil)
        let process = try spawned.get()
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        kill(process.pid, SIGTERM)
        #expect(try await ReapedStatus.await(process) == .signaled(SIGTERM))
    }

    private func sleeper() throws -> SpawnedProcess {
        try PosixSpawner.spawn(
            SpawnRequest(executable: "/bin/sleep", arguments: ["30"], environment: EnvironmentList([])))
    }
}
