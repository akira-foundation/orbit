import Foundation

public final class SpawnedProcess: Sendable {
    public let pid: Int32
    public let stdout: FileHandle?
    public let stderr: FileHandle?
    private let reaper: ProcessReaper

    init(pid: Int32, stdout: FileHandle?, stderr: FileHandle?) {
        self.pid = pid
        self.stdout = stdout
        self.stderr = stderr
        reaper = ProcessReaper(pid: pid)
    }

    public func exitStatus() async -> ExitStatus {
        await reaper.exitStatus()
    }
}
