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

let arguments = Array(CommandLine.arguments.dropFirst())
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
