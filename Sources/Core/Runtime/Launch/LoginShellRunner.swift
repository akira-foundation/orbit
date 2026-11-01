import Darwin
import Dispatch
import Foundation

public struct LoginShellRunner: LoginShellRunning {
    private static let pollSlice: Duration = .milliseconds(10)
    private static let terminationGrace: Duration = .milliseconds(100)

    public init() {}

    public func run(shell: String, script: String, timeout: Duration) async -> String? {
        await withCheckedContinuation { continuation in
            DispatchQueue.global().async {
                continuation.resume(returning: Self.execute(shell: shell, script: script, timeout: timeout))
            }
        }
    }

    private static func execute(shell: String, script: String, timeout: Duration) -> String? {
        var pipeFds: [Int32] = [0, 0]
        guard pipe(&pipeFds) == 0 else {
            return nil
        }
        let readFd = pipeFds[0]
        let writeFd = pipeFds[1]
        defer { close(readFd) }
        _ = fcntl(readFd, F_SETFD, FD_CLOEXEC)
        _ = fcntl(writeFd, F_SETFD, FD_CLOEXEC)
        let spawned = spawn(shell: shell, arguments: ["-lic", script], standardOutput: writeFd)
        close(writeFd)
        guard let pid = spawned else {
            return nil
        }
        return collect(from: readFd, child: pid, deadline: ContinuousClock.now + timeout)
    }

    private static func spawn(shell: String, arguments: [String], standardOutput: Int32) -> pid_t? {
        var attributes: posix_spawnattr_t?
        var actions: posix_spawn_file_actions_t?
        posix_spawnattr_init(&attributes)
        posix_spawn_file_actions_init(&actions)
        defer {
            posix_spawnattr_destroy(&attributes)
            posix_spawn_file_actions_destroy(&actions)
        }
        posix_spawnattr_setflags(&attributes, Int16(POSIX_SPAWN_SETPGROUP | POSIX_SPAWN_CLOEXEC_DEFAULT))
        posix_spawnattr_setpgroup(&attributes, 0)
        posix_spawn_file_actions_addopen(&actions, 0, "/dev/null", O_RDONLY, 0)
        posix_spawn_file_actions_adddup2(&actions, standardOutput, 1)
        posix_spawn_file_actions_addopen(&actions, 2, "/dev/null", O_WRONLY, 0)

        let argv = ([shell] + arguments).map { strdup($0) } + [nil]
        let environment = ProcessInfo.processInfo.environment.map { strdup("\($0.key)=\($0.value)") } + [nil]
        defer {
            argv.forEach { free($0) }
            environment.forEach { free($0) }
        }
        var pid: pid_t = 0
        guard posix_spawn(&pid, shell, &actions, &attributes, argv, environment) == 0 else {
            return nil
        }
        return pid
    }

    private static func collect(from readFd: Int32, child pid: pid_t, deadline: ContinuousClock.Instant) -> String? {
        var output = Data()
        var buffer = [UInt8](repeating: 0, count: 65536)
        var status: Int32 = 0
        while ContinuousClock.now < deadline {
            let readable = waitReadable(readFd, milliseconds: 10)
            if readable {
                let count = read(readFd, &buffer, buffer.count)
                if count > 0 {
                    output.append(contentsOf: buffer[0..<count])
                    continue
                }
            }
            if waitpid(pid, &status, WNOHANG) == pid {
                drain(readFd, into: &output, buffer: &buffer)
                return exitedCleanly(status) ? String(decoding: output, as: UTF8.self) : nil
            }
        }
        terminateGroup(pid)
        return nil
    }

    private static func waitReadable(_ fd: Int32, milliseconds: Int32) -> Bool {
        var descriptor = pollfd(fd: fd, events: Int16(POLLIN), revents: 0)
        return poll(&descriptor, 1, milliseconds) > 0
    }

    private static func drain(_ fd: Int32, into output: inout Data, buffer: inout [UInt8]) {
        while waitReadable(fd, milliseconds: 0) {
            let count = read(fd, &buffer, buffer.count)
            if count <= 0 {
                return
            }
            output.append(contentsOf: buffer[0..<count])
        }
    }

    private static func exitedCleanly(_ status: Int32) -> Bool {
        status & 0x7f == 0 && (status >> 8) & 0xff == 0
    }

    private static func terminateGroup(_ pid: pid_t) {
        var victims = descendants(of: pid)
        kill(-pid, SIGTERM)
        let graceEnd = ContinuousClock.now + terminationGrace
        var status: Int32 = 0
        var reaped = false
        while ContinuousClock.now < graceEnd && !reaped {
            reaped = waitpid(pid, &status, WNOHANG) == pid
            usleep(5000)
        }
        victims.formUnion(descendants(of: pid))
        kill(-pid, SIGKILL)
        victims.forEach { kill($0, SIGKILL) }
        if !reaped {
            waitpid(pid, &status, 0)
        }
    }

    private static func descendants(of pid: pid_t) -> Set<pid_t> {
        var found: Set<pid_t> = []
        var pending = [pid]
        while let parent = pending.popLast() {
            for child in children(of: parent) where found.insert(child).inserted {
                pending.append(child)
            }
        }
        return found
    }

    private static func children(of pid: pid_t) -> [pid_t] {
        let capacity = 256
        var buffer = [pid_t](repeating: 0, count: capacity)
        let count = proc_listchildpids(pid, &buffer, Int32(capacity * MemoryLayout<pid_t>.size))
        guard count > 0 else {
            return []
        }
        return Array(buffer.prefix(Int(count)))
    }
}
