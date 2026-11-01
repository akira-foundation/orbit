import Darwin

@testable import Core

struct SpawnedTestProcess {
    let pid: pid_t

    static func sleep() throws -> SpawnedTestProcess {
        var attributes: posix_spawnattr_t?
        posix_spawnattr_init(&attributes)
        defer { posix_spawnattr_destroy(&attributes) }
        posix_spawnattr_setflags(&attributes, Int16(POSIX_SPAWN_SETPGROUP))
        posix_spawnattr_setpgroup(&attributes, 0)
        let arguments: [UnsafeMutablePointer<CChar>?] = [strdup("/bin/sleep"), strdup("30"), nil]
        defer { arguments.forEach { free($0) } }
        let environment: [UnsafeMutablePointer<CChar>?] = [nil]
        var pid: pid_t = 0
        let status = posix_spawn(&pid, "/bin/sleep", nil, &attributes, arguments, environment)
        guard status == 0 else { throw RuntimeError.message("spawn /bin/sleep: \(status)") }
        return SpawnedTestProcess(pid: pid)
    }

    func terminate() {
        Darwin.kill(-pid, SIGKILL)
        Darwin.kill(pid, SIGKILL)
        var status: Int32 = 0
        while waitpid(pid, &status, 0) == -1 && errno == EINTR {}
    }
}
