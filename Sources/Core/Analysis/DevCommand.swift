enum DevCommand {
    static let preferredScripts = ["dev", "start:dev", "develop", "serve", "start"]

    static func suggest(scripts: [String: String], packageManager: String) -> String {
        let name = preferredScripts.first { scripts[$0] != nil } ?? "dev"
        return switch packageManager {
        case "yarn": "yarn \(name)"
        case "bun": "bun run \(name)"
        default: "\(packageManager) run \(name)"
        }
    }

    static func port(in scripts: [String: String]) -> Int? {
        for name in preferredScripts {
            guard let command = scripts[name] else {
                continue
            }
            for flag in ["--port ", "--port="] {
                if let port = leadingNumber(after: flag, in: command) {
                    return port
                }
            }
        }
        return nil
    }

    private static func leadingNumber(after flag: String, in command: String) -> Int? {
        guard let range = command.range(of: flag) else {
            return nil
        }
        let digits = command[range.upperBound...]
            .drop { $0.isWhitespace }
            .prefix { $0.isASCII && $0.isNumber }
        guard let port = Int(digits), port > 0 else {
            return nil
        }
        return port
    }
}
