public enum SupervisorArguments {
    public static let mode = "--runtime-supervisor"
    private static let separator = "--"
    private static let options = ["--session", "--token", "--control-fd", "--cwd"]

    public static func make(_ invocation: SupervisorInvocation) -> [String] {
        [
            mode, "--session", invocation.sessionID, "--token", invocation.token, "--control-fd",
            String(invocation.controlDescriptor), "--cwd", invocation.directory, separator,
        ] + invocation.command
    }

    public static func parse(_ arguments: [String]) throws -> SupervisorInvocation {
        guard arguments.first == mode else { throw RuntimeError.message("runtime: supervisor mode missing") }
        guard let end = arguments.firstIndex(of: separator), end < arguments.count - 1 else {
            throw RuntimeError.message("runtime: supervisor command missing")
        }
        let values = try values(Array(arguments[1..<end]))
        guard let descriptor = values["--control-fd"].flatMap({ Int32($0) }), descriptor >= 3 else {
            throw RuntimeError.message("runtime: invalid supervisor control descriptor")
        }
        let directory = values["--cwd"] ?? ""
        guard directory.hasPrefix("/") else { throw RuntimeError.message("runtime: supervisor cwd must be absolute") }
        let sessionID = values["--session"] ?? ""
        let token = values["--token"] ?? ""
        guard isHex(sessionID, length: 16), isHex(token, length: 32) else {
            throw RuntimeError.message("runtime: invalid supervisor ownership identifiers")
        }
        let command = Array(arguments[(end + 1)...])
        guard command[0].hasPrefix("/") else {
            throw RuntimeError.message("runtime: supervisor command must be an absolute path")
        }
        return SupervisorInvocation(
            sessionID: sessionID, token: token, controlDescriptor: descriptor, directory: directory, command: command)
    }

    private static func values(_ pairs: [String]) throws -> [String: String] {
        guard pairs.count.isMultiple(of: 2) else { throw RuntimeError.message("runtime: malformed supervisor options") }
        var values: [String: String] = [:]
        for index in stride(from: 0, to: pairs.count, by: 2) {
            let name = pairs[index]
            guard options.contains(name) else {
                throw RuntimeError.message("runtime: unknown supervisor option \"\(name)\"")
            }
            guard values[name] == nil, !pairs[index + 1].isEmpty else {
                throw RuntimeError.message("runtime: duplicate or empty supervisor option \"\(name)\"")
            }
            values[name] = pairs[index + 1]
        }
        if let missing = options.first(where: { values[$0] == nil }) {
            throw RuntimeError.message("runtime: missing supervisor option \"\(missing)\"")
        }
        return values
    }

    private static func isHex(_ identifier: String, length: Int) -> Bool {
        identifier.utf8.count == length
            && identifier.utf8.allSatisfy {
                (48...57).contains($0) || (65...70).contains($0) || (97...102).contains($0)
            }
    }
}
