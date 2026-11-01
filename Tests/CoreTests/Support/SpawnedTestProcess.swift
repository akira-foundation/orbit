import Darwin

@testable import Core

struct SpawnedTestProcess {
    let pid: pid_t

    static func sleep() throws -> SpawnedTestProcess {
        try spawn("/bin/sleep", ["30"])
    }

    static func spawn(_ path: String, _ arguments: [String]) throws -> SpawnedTestProcess {
        var attributes: posix_spawnattr_t?
        posix_spawnattr_init(&attributes)
        defer { posix_spawnattr_destroy(&attributes) }
        posix_spawnattr_setflags(&attributes, Int16(POSIX_SPAWN_SETPGROUP | POSIX_SPAWN_CLOEXEC_DEFAULT))
        posix_spawnattr_setpgroup(&attributes, 0)
        var actions: posix_spawn_file_actions_t?
        posix_spawn_file_actions_init(&actions)
        defer { posix_spawn_file_actions_destroy(&actions) }
        posix_spawn_file_actions_addopen(&actions, 1, "/dev/null", O_WRONLY, 0)
        posix_spawn_file_actions_addopen(&actions, 2, "/dev/null", O_WRONLY, 0)
        posix_spawn_file_actions_addopen(&actions, 0, "/dev/null", O_RDONLY, 0)
        let argv: [UnsafeMutablePointer<CChar>?] = ([path] + arguments).map { strdup($0) } + [nil]
        defer { argv.forEach { free($0) } }
        let environment: [UnsafeMutablePointer<CChar>?] = [nil]
        var pid: pid_t = 0
        let status = posix_spawn(&pid, path, &actions, &attributes, argv, environment)
        guard status == 0 else { throw RuntimeError.message("spawn \(path): \(status)") }
        return SpawnedTestProcess(pid: pid)
    }

    func terminate() {
        Darwin.kill(-pid, SIGKILL)
        Darwin.kill(pid, SIGKILL)
        var status: Int32 = 0
        while waitpid(pid, &status, 0) == -1 && errno == EINTR {}
    }
}
