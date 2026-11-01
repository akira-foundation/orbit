import Core
import Foundation

func launchSupervisor(_ command: [String]) throws {
    let executable = Bundle.main.executablePath ?? CommandLine.arguments[0]
    let handle = try SupervisorLauncher(executable: executable, inspector: DarwinProcessInspector()).launch(
        SupervisorSpec(
            sessionID: OwnershipIdentifiers.sessionID(), token: OwnershipIdentifiers.token(),
            directory: FileManager.default.currentDirectoryPath, command: command, grace: .milliseconds(20)))
    print("SUPERVISOR_PID=\(handle.identity.pid)")
    if let line = handle.stdout.flatMap(firstLine) {
        print(line)
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

let arguments = Array(CommandLine.arguments.dropFirst())
if let code = RuntimeSupervisor.runIfRequested(arguments: arguments) {
    exit(code)
}
switch arguments.first {
case "--launch-supervisor" where arguments.count > 1:
    do {
        try launchSupervisor(Array(arguments.dropFirst()))
    } catch {
        FileHandle.standardError.write(Data("\(error)\n".utf8))
        exit(1)
    }
case "--print-environment":
    for (key, value) in ProcessInfo.processInfo.environment.sorted(by: { $0.key < $1.key }) {
        print("\(key)=\(value)")
    }
default:
    FileHandle.standardError.write(Data("usage: orbit-runtime-fixture --launch-supervisor <command...>\n".utf8))
    exit(2)
}
