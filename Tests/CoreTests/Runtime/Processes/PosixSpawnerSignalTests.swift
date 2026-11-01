import Darwin
import Testing

@testable import Core

@Suite(.serialized)
struct PosixSpawnerSignalTests {
    @Test func signalIgnoredByTheParentIsDefaultInTheChild() async throws {
        let previous = signal(SIGTERM, SIG_IGN)
        let spawned = Result { try sleeper() }
        signal(SIGTERM, previous)
        let process = try spawned.get()
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        kill(process.pid, SIGTERM)
        #expect(try await ReapedStatus.await(process) == .signaled(SIGTERM))
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
