import Darwin
import Foundation
import Testing

@testable import Core

struct PosixSpawnerTests {
    @Test func spawnRunsInDirectoryWithEnvironmentAndReportsExit() async throws {
        let directory = try TemporaryDirectory()
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", "pwd; echo \"$X\"; exit 3"],
                environment: EnvironmentList(["X=y"]), directory: directory.path, captureOutput: true))
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        #expect(try fixture.outputLines() == [try canonical(directory.path), "y"])
        #expect(await process.exitStatus() == .exited(3))
    }

    @Test func newLeaderHasItsOwnGroup() async throws {
        let process = try sleeper(group: .newLeader)
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        #expect(getpgid(process.pid) == process.pid)
    }

    @Test func joinPlacesChildInExistingGroup() async throws {
        let leader = try sleeper(group: .newLeader)
        let fixture = SpawnedTestProcess(adopting: leader)
        defer { fixture.terminate() }
        let member = try sleeper(group: .join(leader.pid))
        #expect(getpgid(member.pid) == leader.pid)
        fixture.terminate()
        #expect(await member.exitStatus() == .signaled(SIGKILL))
    }

    @Test func inheritKeepsTheParentGroupAndReadsTheGivenStandardInput() async throws {
        let input = try PosixSpawner.makePipe()
        var writeEnd: Int32? = input.write
        defer {
            close(input.read)
            if let writeEnd { close(writeEnd) }
        }
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", "read _; exit 0"], environment: EnvironmentList([]),
                group: .inherit, standardInput: input.read))
        #expect(getpgid(process.pid) == getpgrp())
        close(input.write)
        writeEnd = nil
        #expect(await process.exitStatus() == .exited(0))
    }

    @Test func spawnedChildInheritsOnlyStandardDescriptorsAndFD3() async throws {
        let leaked = try leakedPipe()
        defer { leaked.forEach { close($0) } }
        let control = try PosixSpawner.makePipe()
        defer {
            close(control.read)
            close(control.write)
        }
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", "ls /dev/fd"], environment: EnvironmentList([]),
                captureOutput: true, controlDescriptor: control.read))
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        let descriptors = Set(try fixture.outputLines())
        let standard: Set<String> = ["0", "1", "2", "3"]
        #expect(standard.isSubset(of: descriptors))
        #expect(descriptors.subtracting(standard).count <= 2)
        #expect(descriptors.isDisjoint(with: leaked.map(String.init)))
        #expect(await process.exitStatus() == .exited(0))
    }

    @Test func controlDescriptorArrivesAsFD3() async throws {
        let control = try PosixSpawner.makePipe()
        defer { close(control.read) }
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", "echo control >&3"], environment: EnvironmentList([]),
                controlDescriptor: control.write))
        close(control.write)
        #expect(await process.exitStatus() == .exited(0))
        let output = FileHandle(fileDescriptor: control.read, closeOnDealloc: false).readDataToEndOfFile()
        #expect(String(decoding: output, as: UTF8.self) == "control\n")
    }

    @Test func withoutCaptureThereAreNoOutputHandles() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(executable: "/bin/sh", arguments: ["-c", "exit 0"], environment: EnvironmentList([])))
        #expect(process.stdout == nil)
        #expect(process.stderr == nil)
        #expect(await process.exitStatus() == .exited(0))
    }

    @Test func standardErrorIsCapturedSeparately() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", "echo out; echo err >&2"], environment: EnvironmentList([]),
                captureOutput: true))
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        let error = try #require(process.stderr).readDataToEndOfFile()
        #expect(try fixture.outputLines() == ["out"])
        #expect(String(decoding: error, as: UTF8.self) == "err\n")
        #expect(await process.exitStatus() == .exited(0))
    }

    @Test func signaledChildReportsSignal() async throws {
        let process = try sleeper(group: .newLeader)
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        kill(process.pid, SIGKILL)
        #expect(await process.exitStatus() == .signaled(SIGKILL))
    }

    @Test func exitStatusServesSeveralCallersBeforeAndAfterExit() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(executable: "/bin/sh", arguments: ["-c", "exit 7"], environment: EnvironmentList([])))
        async let first = process.exitStatus()
        async let second = process.exitStatus()
        #expect(await [first, second] == [.exited(7), .exited(7)])
        #expect(await process.exitStatus() == .exited(7))
    }

    @Test func childIsReapedWithoutAnyoneWaiting() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(executable: "/bin/sh", arguments: ["-c", "exit 0"], environment: EnvironmentList([])))
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(5))
        while kill(process.pid, 0) == 0 && clock.now < deadline {
            try await clock.sleep(for: .milliseconds(5))
        }
        #expect(kill(process.pid, 0) == -1 && errno == ESRCH)
    }

    @Test func missingExecutableFailsWithStartError() {
        #expect(throws: RuntimeError.message("start: No such file or directory")) {
            try PosixSpawner.spawn(
                SpawnRequest(executable: "/nonexistent/orbit-missing", environment: EnvironmentList([])))
        }
    }

    @Test func makePipeMarksBothEndsCloseOnExec() throws {
        let ends = try PosixSpawner.makePipe()
        defer {
            close(ends.read)
            close(ends.write)
        }
        #expect(fcntl(ends.read, F_GETFD) & FD_CLOEXEC != 0)
        #expect(fcntl(ends.write, F_GETFD) & FD_CLOEXEC != 0)
    }

    private func sleeper(group: ProcessGroupPlacement) throws -> SpawnedProcess {
        try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sleep", arguments: ["30"], environment: EnvironmentList([]), group: group))
    }

    private func leakedPipe() throws -> [Int32] {
        var ends: [Int32] = [0, 0]
        guard pipe(&ends) == 0 else { throw RuntimeError.message("pipe failed") }
        defer { ends.forEach { close($0) } }
        let high = ends.map { fcntl($0, F_DUPFD, 200) }
        guard high.allSatisfy({ $0 >= 200 }) else { throw RuntimeError.message("dup failed") }
        return high
    }

    private func canonical(_ path: String) throws -> String {
        guard let resolved = realpath(path, nil) else { throw RuntimeError.message("realpath \(path)") }
        defer { free(resolved) }
        return String(cString: resolved)
    }
}
