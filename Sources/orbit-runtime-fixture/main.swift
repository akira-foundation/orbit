import Core
import Darwin
import Foundation

func launchSupervisor(_ arguments: [String]) async throws {
    let holdsControl = arguments.first == "--hold-control"
    let command = holdsControl ? Array(arguments.dropFirst()) : arguments
    let executable = Bundle.main.executablePath ?? CommandLine.arguments[0]
    let handle = try await SupervisorLauncher(executable: executable, inspector: DarwinProcessInspector()).launch(
        SupervisorSpec(
            sessionID: OwnershipIdentifiers.sessionID(), token: OwnershipIdentifiers.token(),
            directory: FileManager.default.currentDirectoryPath, command: command, grace: .milliseconds(20)))
    print("SUPERVISOR_PID=\(handle.identity.pid)")
    if let line = handle.stdout.flatMap(firstLine) {
        print(line)
    }
    try await waitForSleepingMembers(of: handle.identity)
    if holdsControl {
        print("HOLDER_PID=\(try holdControl(of: handle))")
    }
    fflush(stdout)
    exit(0)
}

func firstLine(_ handle: FileHandle) -> String? {
    var bytes: [UInt8] = []
    while let chunk = try? handle.read(upToCount: 1), let byte = chunk.first {
        if byte == UInt8(ascii: "\n") {
            return String(decoding: bytes, as: UTF8.self)
        }
        bytes.append(byte)
    }
    return nil
}

func waitForSleepingMembers(of supervisor: ProcessIdentity) async throws {
    let clock = ContinuousClock()
    let deadline = clock.now.advanced(by: .seconds(5))
    while clock.now < deadline {
        let members = ((try? DarwinProcessInspector().groupMembers(supervisor.pgid)) ?? []).filter {
            $0.pid != supervisor.pid
        }
        if !members.isEmpty && members.allSatisfy({ $0.arguments.hasPrefix("/bin/sleep ") }) {
            return
        }
        try await clock.sleep(for: .milliseconds(2))
    }
    throw RuntimeError.message("supervised group \(supervisor.pgid) never settled")
}

func holdControl(of handle: any SupervisorHandle) throws -> Int32 {
    let outputs = Set([handle.stdout, handle.stderr].compactMap { $0?.fileDescriptor })
    let control = (3..<getdtablesize()).first { descriptor in
        var status = stat()
        guard !outputs.contains(descriptor), fstat(descriptor, &status) == 0 else { return false }
        return status.st_mode & S_IFMT == S_IFIFO && fcntl(descriptor, F_GETFL) & O_ACCMODE == O_WRONLY
    }
    guard let control else { throw RuntimeError.message("no supervisor control descriptor") }
    let holder = try PosixSpawner.spawn(
        SpawnRequest(
            executable: "/bin/sleep", arguments: ["60"], environment: EnvironmentList([]),
            captureOutput: true, controlDescriptor: control))
    return holder.pid
}

func clobberArgumentsAndHold() -> Never {
    var mib: [Int32] = [CTL_KERN, KERN_PROCARGS2, getpid()]
    var buffer = [UInt8](repeating: 0, count: 1 << 20)
    var size = buffer.count
    guard sysctl(&mib, UInt32(mib.count), &buffer, &size, nil, 0) == 0, let first = CommandLine.unsafeArgv[0] else {
        exit(1)
    }
    var offset = MemoryLayout<Int32>.size
    while offset < size, buffer[offset] != 0 { offset += 1 }
    while offset < size, buffer[offset] == 0 { offset += 1 }
    memset(first, 0x41, size - offset)
    holdWhileParentLives()
}

func leaveTheGroupOnTerm() -> Never {
    signal(SIGTERM) { _ in _ = setsid() }
    print("READY")
    fflush(stdout)
    holdWhileParentLives()
}

func joinGroupAndHold(_ group: Int32) -> Never {
    print(setpgid(0, group) == 0 ? "JOIN_ERRNO=0" : "JOIN_ERRNO=\(errno)")
    fflush(stdout)
    holdWhileParentLives()
}

func printEnvironmentAndHold(_ name: String) -> Never {
    print("\(name)=\(getenv(name).map { String(cString: $0) } ?? "")")
    fflush(stdout)
    holdWhileParentLives()
}

func holdWhileParentLives() -> Never {
    let parent = getppid()
    while getppid() == parent {
        usleep(10_000)
    }
    exit(0)
}

let arguments = Array(CommandLine.arguments.dropFirst())
switch arguments.first {
case "--clobber-arguments": clobberArgumentsAndHold()
case "--setsid-on-term": leaveTheGroupOnTerm()
case "--join-group": joinGroupAndHold(Int32(arguments[1]) ?? 0)
case "--print-environment": printEnvironmentAndHold(arguments[1])
case "--control-on-fd-3": try await writeThroughControlOnDescriptorThree()
default: break
}
if let code = RuntimeSupervisor.runIfRequested(arguments: arguments) {
    exit(code)
}
guard arguments.first == "--launch-supervisor", arguments.count > 1 else {
    try? FileHandle.standardError.write(
        contentsOf: Data("usage: orbit-runtime-fixture --launch-supervisor [--hold-control] <command...>\n".utf8))
    exit(2)
}
do {
    try await launchSupervisor(Array(arguments.dropFirst()))
} catch {
    try? FileHandle.standardError.write(contentsOf: Data("\(error)\n".utf8))
    exit(1)
}
