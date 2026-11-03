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
        #expect(try await process.exitStatus() == .exited(3))
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
        #expect(try await member.exitStatus() == .signaled(SIGKILL))
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
        #expect(try await process.exitStatus() == .exited(0))
    }

    @Test func spawnedChildInheritsOnlyStandardDescriptorsAndFD3() async throws {
        let leaked = try leakedPipe()
        defer { leaked.forEach { close($0) } }
        let control = try PosixSpawner.makePipe()
        defer {
            close(control.read)
            close(control.write)
        }
        #expect(try await openDescriptors(control: control.read) == ["0", "1", "2", "3"])
        #expect(try await openDescriptors(control: nil) == ["0", "1", "2"])
    }

    @Test func controlDescriptorAlreadyOnFD3StillReachesTheChild() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: ProcessFixture.runtimeFixturePath, arguments: ["--control-on-fd-3"],
                environment: EnvironmentList([]), captureOutput: true))
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        #expect(try fixture.outputLines() == ["control"])
        #expect(try await process.exitStatus() == .exited(0))
    }

    @Test func controlFromAStandardDescriptorIsNotOverwrittenByTheRedirectedInput() async throws {
        let input = try PosixSpawner.makePipe()
        var writeEnd: Int32? = input.write
        defer {
            close(input.read)
            if let writeEnd { close(writeEnd) }
        }
        let probe = "read line; [ \"$line\" = in ] || exit 2; [ /dev/fd/0 -ef /dev/fd/3 ] && exit 3; exit 0"
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", probe], environment: EnvironmentList([]),
                controlDescriptor: STDIN_FILENO, standardInput: input.read))
        _ = "in\n".withCString { write(input.write, $0, 3) }
        close(input.write)
        writeEnd = nil
        #expect(try await process.exitStatus() == .exited(0))
    }

    @Test func invalidControlDescriptorFailsWithStartError() {
        #expect(throws: RuntimeError.message("start: Bad file descriptor")) {
            try PosixSpawner.spawn(
                SpawnRequest(executable: "/bin/sh", environment: EnvironmentList([]), controlDescriptor: -1))
        }
    }

    @Test func controlDescriptorArrivesAsFD3() async throws {
        let control = try PosixSpawner.makePipe()
        defer { close(control.read) }
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", "echo control >&3"], environment: EnvironmentList([]),
                controlDescriptor: control.write))
        close(control.write)
        #expect(try await process.exitStatus() == .exited(0))
        let output = FileHandle(fileDescriptor: control.read, closeOnDealloc: false).readDataToEndOfFile()
        #expect(String(decoding: output, as: UTF8.self) == "control\n")
    }

    @Test func withoutCaptureThereAreNoOutputHandles() async throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(executable: "/bin/sh", arguments: ["-c", "exit 0"], environment: EnvironmentList([])))
        #expect(process.stdout == nil)
        #expect(process.stderr == nil)
        #expect(try await process.exitStatus() == .exited(0))
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
        #expect(try await process.exitStatus() == .exited(0))
    }

    @Test func signaledChildReportsSignal() async throws {
        let process = try sleeper(group: .newLeader)
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        kill(process.pid, SIGKILL)
        #expect(try await process.exitStatus() == .signaled(SIGKILL))
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

    private func openDescriptors(control: Int32?) async throws -> [String] {
        let probe = "i=0; while [ $i -lt 300 ]; do [ -e /dev/fd/$i ] && echo $i; i=$((i + 1)); done; exit 0"
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", probe], environment: EnvironmentList([]),
                captureOutput: true, controlDescriptor: control))
        let fixture = SpawnedTestProcess(adopting: process)
        defer { fixture.terminate() }
        let lines = try fixture.outputLines()
        #expect(try await process.exitStatus() == .exited(0))
        return lines
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
