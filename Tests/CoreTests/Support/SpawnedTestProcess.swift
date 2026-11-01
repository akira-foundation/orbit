import Darwin
import Foundation

@testable import Core

struct SpawnedTestProcess {
    private static let lineTimeout: Duration = .seconds(5)

    let pid: pid_t
    private let output: FileHandle?
    private let reapedByCore: Bool

    init(pid: pid_t) {
        self.pid = pid
        output = nil
        reapedByCore = false
    }

    init(adopting process: SpawnedProcess) {
        pid = process.pid
        output = process.stdout
        reapedByCore = true
    }

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
        guard !reapedByCore else { return }
        Darwin.kill(pid, SIGKILL)
        var status: Int32 = 0
        while waitpid(pid, &status, 0) == -1 && errno == EINTR {}
    }

    func waitForLine(_ expected: String) throws {
        var scanner = LineScanner()
        guard try read(until: { bytes in scanner.feed(bytes).contains(expected) }) else {
            throw RuntimeError.message("PID \(pid) closed its output before \"\(expected)\"")
        }
    }

    func outputLines() throws -> [String] {
        var scanner = LineScanner()
        var lines: [String] = []
        _ = try read { bytes in
            lines += scanner.feed(bytes)
            return false
        }
        return lines + scanner.finish()
    }

    private func read(until done: (ArraySlice<UInt8>) -> Bool) throws -> Bool {
        guard let descriptor = output?.fileDescriptor else { throw RuntimeError.message("no captured output") }
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: Self.lineTimeout)
        var buffer = [UInt8](repeating: 0, count: 4096)
        while clock.now < deadline {
            var poller = pollfd(fd: descriptor, events: Int16(POLLIN), revents: 0)
            guard poll(&poller, 1, 10) > 0 else { continue }
            let count = Darwin.read(descriptor, &buffer, buffer.count)
            if count <= 0 {
                return false
            }
            if done(buffer[0..<count]) {
                return true
            }
        }
        throw RuntimeError.message("output of PID \(pid) did not finish in time")
    }
}
