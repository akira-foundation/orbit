import Darwin
import Testing

@testable import Core

struct ProcessReaperTests {
    @Test func exitStatusServesWaitersRegisteredBeforeExitAndCallersAfterIt() async throws {
        let input = try PosixSpawner.makePipe()
        var writeEnd: Int32? = input.write
        defer {
            close(input.read)
            if let writeEnd { close(writeEnd) }
        }
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", "read _; exit 7"], environment: EnvironmentList([]),
                standardInput: input.read))
        async let first = process.exitStatus()
        async let second = process.exitStatus()
        try await waitForWaiters(2, on: process)
        close(input.write)
        writeEnd = nil
        #expect(try await [first, second] == [.exited(7), .exited(7)])
        #expect(try await process.exitStatus() == .exited(7))
    }

    @Test func cancelledWaiterReturnsAndTheProcessIsStillReaped() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(executable: "/bin/sleep", arguments: ["30"], environment: EnvironmentList([])))
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        let waiter = Task { try await process.exitStatus() }
        try await waitForWaiters(1, on: process)
        waiter.cancel()
        await #expect(throws: RuntimeError.cancelled) { try await waiter.value }
        #expect(process.reaper.waiterCount == 0)
        kill(process.pid, SIGKILL)
        #expect(try await process.exitStatus() == .signaled(SIGKILL))
    }

    @Test func alreadyCancelledWaiterReturnsWithoutWaiting() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(executable: "/bin/sleep", arguments: ["30"], environment: EnvironmentList([])))
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        let waiter = Task {
            withUnsafeCurrentTask { $0?.cancel() }
            return try await process.exitStatus()
        }
        await #expect(throws: RuntimeError.cancelled) { try await waiter.value }
        #expect(process.reaper.waiterCount == 0)
    }

    @Test func waitpidFailureIsUnknownRatherThanAnExit() async throws {
        let reaper = ProcessReaper(pid: 1)
        #expect(try await reaper.exitStatus() == .unknown(ECHILD))
    }

    private func waitForWaiters(_ count: Int, on process: SpawnedProcess) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(5))
        while process.reaper.waiterCount < count {
            guard clock.now < deadline else { throw RuntimeError.message("waiters never registered") }
            try await clock.sleep(for: .milliseconds(2))
        }
    }
}
