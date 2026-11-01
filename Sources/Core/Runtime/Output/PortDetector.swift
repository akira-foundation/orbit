public enum PortDetector {
    nonisolated(unsafe) private static let conflict = /in use|EADDRINUSE|trying another/.ignoresCase()

    nonisolated(unsafe) private static let hostPort =
        /(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1\])[: ]([0-9]{2,5})/.ignoresCase()

    nonisolated(unsafe) private static let portFlag = /\bport[:= ]\s*([0-9]{2,5})\b/.ignoresCase()
        .asciiOnlyCharacterClasses()

    nonisolated(unsafe) private static let endpoint =
        /https?:\/\/(?:\[[0-9a-f:]+\]|[a-z0-9.-]+):[0-9]{2,5}/.ignoresCase()

    public static func port(in line: String) -> DetectedPort? {
        let text = ANSI.strip(line)
        if text.contains(conflict) {
            return nil
        }
        if let port = text.firstMatch(of: hostPort).flatMap({ validPort($0.1) }) {
            return DetectedPort(port: port, strong: true)
        }
        if let port = text.firstMatch(of: portFlag).flatMap({ validPort($0.1) }) {
            return DetectedPort(port: port, strong: false)
        }
        return nil
    }

    public static func httpEndpoint(in line: String) -> HTTPEndpoint? {
        guard let match = ANSI.strip(line).firstMatch(of: endpoint) else {
            return nil
        }
        let text = String(match.0)
        guard let schemeEnd = text.range(of: "://"), let portSeparator = text.lastIndex(of: ":"),
            let port = validPort(text[text.index(after: portSeparator)...])
        else {
            return nil
        }
        let host = text[schemeEnd.upperBound..<portSeparator].filter { $0 != "[" && $0 != "]" }
        return HTTPEndpoint(
            scheme: text[..<schemeEnd.lowerBound].lowercased(), host: host.lowercased(), port: port)
    }

    private static func validPort(_ digits: Substring) -> Int? {
        guard let port = Int(digits), (1..<65536).contains(port) else {
            return nil
        }
        return port
    }
}
