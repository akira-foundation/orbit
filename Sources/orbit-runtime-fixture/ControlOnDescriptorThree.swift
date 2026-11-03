import Core
import Darwin
import Foundation

func writeThroughControlOnDescriptorThree() async throws -> Never {
    let target: Int32 = 3
    let control = try PosixSpawner.makePipe()
    guard control.read == target else { throw RuntimeError.message("descriptor \(target) was not free") }
    let readEnd = fcntl(control.read, F_DUPFD_CLOEXEC, 10)
    guard readEnd >= 0, dup2(control.write, target) == target, fcntl(target, F_SETFD, FD_CLOEXEC) == 0 else {
        throw RuntimeError.message("could not place control on descriptor \(target)")
    }
    close(control.write)
    let process = try PosixSpawner.spawn(
        SpawnRequest(
            executable: "/bin/sh", arguments: ["-c", "echo control >&3"], environment: EnvironmentList([]),
            controlDescriptor: target))
    close(target)
    let status = try await process.exitStatus()
    guard status == .exited(0) else { throw RuntimeError.message("control child ended with \(status)") }
    FileHandle.standardOutput.write(FileHandle(fileDescriptor: readEnd).readDataToEndOfFile())
    exit(0)
}
