import Darwin
import Foundation

public enum PosixSpawner {
    private static let controlTarget: Int32 = 3
    private static let highDescriptorFloor: Int32 = 10

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
        var raised: [Int32] = []
        defer { raised.forEach { close($0) } }
        let raise = { (descriptor: Int32?) throws -> Int32? in
            guard let descriptor, descriptor <= controlTarget else { return descriptor }
            let copy = fcntl(descriptor, F_DUPFD_CLOEXEC, highDescriptorFloor)
            guard copy >= 0 else { throw startError(errno) }
            raised.append(copy)
            return copy
        }
        let input = try raise(request.standardInput)
        let routedOutput = try raise(output)
        let routedErrors = try raise(errors)
        let control = try raise(request.controlDescriptor)

        var attributes: posix_spawnattr_t?
        var actions: posix_spawn_file_actions_t?
        try check(posix_spawnattr_init(&attributes))
        defer { posix_spawnattr_destroy(&attributes) }
        try check(posix_spawn_file_actions_init(&actions))
        defer { posix_spawn_file_actions_destroy(&actions) }
        try configure(&attributes, group: request.group)
        if let input {
            try check(posix_spawn_file_actions_adddup2(&actions, input, STDIN_FILENO))
        } else {
            try check(posix_spawn_file_actions_addopen(&actions, STDIN_FILENO, "/dev/null", O_RDONLY, 0))
        }
        try route(routedOutput, to: STDOUT_FILENO, &actions)
        try route(routedErrors, to: STDERR_FILENO, &actions)
        if let control {
            try check(posix_spawn_file_actions_adddup2(&actions, control, controlTarget))
        }
        if let directory = request.directory {
            try check(posix_spawn_file_actions_addchdir(&actions, directory))
        }
        let argv = ([request.executable] + request.arguments).map { strdup($0) } + [nil]
        let environment = request.environment.entries.map { strdup($0) } + [nil]
        defer {
            argv.forEach { free($0) }
            environment.forEach { free($0) }
        }
        var pid: pid_t = 0
        try check(posix_spawn(&pid, request.executable, &actions, &attributes, argv, environment))
        return pid
    }

    private static func configure(_ attributes: inout posix_spawnattr_t?, group: ProcessGroupPlacement) throws {
        var flags = Int32(POSIX_SPAWN_CLOEXEC_DEFAULT | POSIX_SPAWN_SETSIGDEF | POSIX_SPAWN_SETSIGMASK)
        switch group {
        case .newLeader:
            flags |= Int32(POSIX_SPAWN_SETPGROUP)
            try check(posix_spawnattr_setpgroup(&attributes, 0))
        case .join(let pgid):
            flags |= Int32(POSIX_SPAWN_SETPGROUP)
            try check(posix_spawnattr_setpgroup(&attributes, pgid))
        case .inherit:
            break
        }
        var defaults = CatchableSignals.all
        var mask = sigset_t()
        try check(posix_spawnattr_setsigdefault(&attributes, &defaults))
        try check(posix_spawnattr_setsigmask(&attributes, &mask))
        try check(posix_spawnattr_setflags(&attributes, Int16(flags)))
    }

    private static func route(
        _ descriptor: Int32?, to target: Int32, _ actions: inout posix_spawn_file_actions_t?
    ) throws {
        guard let descriptor else {
            try check(posix_spawn_file_actions_addinherit_np(&actions, target))
            return
        }
        try check(posix_spawn_file_actions_adddup2(&actions, descriptor, target))
    }

    private static func check(_ code: Int32) throws {
        guard code == 0 else { throw startError(code) }
    }

    private static func startError(_ code: Int32) -> RuntimeError {
        RuntimeError.message("start: \(errorText(code))")
    }

    private static func errorText(_ code: Int32) -> String {
        String(cString: strerror(code))
    }
}
