import Darwin
import Foundation

public enum PosixSpawner {
    private static let controlTarget: Int32 = 3

    public static func spawn(_ request: SpawnRequest) throws -> SpawnedProcess {
        var opened: [Int32] = []
        do {
            let output = request.captureOutput ? try makePipe(context: "stdout pipe") : nil
            output.map { opened += [$0.read, $0.write] }
            let errors = request.captureOutput ? try makePipe(context: "stderr pipe") : nil
            errors.map { opened += [$0.read, $0.write] }
            let pid = try launch(request, output: output?.write, errors: errors?.write)
            [output?.write, errors?.write].compactMap(\.self).forEach { close($0) }
            return SpawnedProcess(
                pid: pid,
                stdout: output.map { FileHandle(fileDescriptor: $0.read, closeOnDealloc: true) },
                stderr: errors.map { FileHandle(fileDescriptor: $0.read, closeOnDealloc: true) })
        } catch {
            opened.forEach { close($0) }
            throw error
        }
    }

    public static func makePipe() throws -> (read: Int32, write: Int32) {
        try makePipe(context: "pipe")
    }

    private static func makePipe(context: String) throws -> (read: Int32, write: Int32) {
        var ends: [Int32] = [-1, -1]
        guard pipe(&ends) == 0 else { throw RuntimeError.message("\(context): \(errorText(errno))") }
        for end in ends where fcntl(end, F_SETFD, FD_CLOEXEC) == -1 {
            let code = errno
            ends.forEach { close($0) }
            throw RuntimeError.message("\(context): \(errorText(code))")
        }
        return (ends[0], ends[1])
    }

    private static func launch(_ request: SpawnRequest, output: Int32?, errors: Int32?) throws -> Int32 {
        var attributes: posix_spawnattr_t?
        var actions: posix_spawn_file_actions_t?
        posix_spawnattr_init(&attributes)
        posix_spawn_file_actions_init(&actions)
        defer {
            posix_spawnattr_destroy(&attributes)
            posix_spawn_file_actions_destroy(&actions)
        }
        configureGroup(request.group, &attributes)
        if let input = request.standardInput {
            posix_spawn_file_actions_adddup2(&actions, input, STDIN_FILENO)
        } else {
            posix_spawn_file_actions_addopen(&actions, STDIN_FILENO, "/dev/null", O_RDONLY, 0)
        }
        route(output, to: STDOUT_FILENO, &actions)
        route(errors, to: STDERR_FILENO, &actions)
        if let control = request.controlDescriptor {
            posix_spawn_file_actions_adddup2(&actions, control, controlTarget)
        }
        if let directory = request.directory {
            posix_spawn_file_actions_addchdir(&actions, directory)
        }
        let argv = ([request.executable] + request.arguments).map { strdup($0) } + [nil]
        let environment = request.environment.entries.map { strdup($0) } + [nil]
        defer {
            argv.forEach { free($0) }
            environment.forEach { free($0) }
        }
        var pid: pid_t = 0
        let status = posix_spawn(&pid, request.executable, &actions, &attributes, argv, environment)
        guard status == 0 else { throw RuntimeError.message("start: \(errorText(status))") }
        return pid
    }

    private static func configureGroup(_ placement: ProcessGroupPlacement, _ attributes: inout posix_spawnattr_t?) {
        var flags = Int32(POSIX_SPAWN_CLOEXEC_DEFAULT)
        switch placement {
        case .newLeader:
            flags |= Int32(POSIX_SPAWN_SETPGROUP)
            posix_spawnattr_setpgroup(&attributes, 0)
        case .join(let pgid):
            flags |= Int32(POSIX_SPAWN_SETPGROUP)
            posix_spawnattr_setpgroup(&attributes, pgid)
        case .inherit:
            break
        }
        posix_spawnattr_setflags(&attributes, Int16(flags))
    }

    private static func route(_ descriptor: Int32?, to target: Int32, _ actions: inout posix_spawn_file_actions_t?) {
        if let descriptor {
            posix_spawn_file_actions_adddup2(&actions, descriptor, target)
        } else {
            posix_spawn_file_actions_addinherit_np(&actions, target)
        }
    }

    private static func errorText(_ code: Int32) -> String {
        String(cString: strerror(code))
    }
}
