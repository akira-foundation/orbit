import Foundation

public enum DotEnv {
    public static let files = [".env", ".env.local", ".env.development", ".env.development.local"]

    private static let maximumLineLength = 64 * 1024
    private static let newline: UInt8 = 10
    private static let carriageReturn: UInt8 = 13

    public static func parse(_ text: String) -> [(key: String, value: String)] {
        var pairs: [(key: String, value: String)] = []
        for bytes in text.utf8.split(separator: newline, omittingEmptySubsequences: false) {
            if bytes.count >= maximumLineLength {
                break
            }
            let content = bytes.last == carriageReturn ? bytes.dropLast() : bytes
            if let pair = pair(from: String(decoding: content, as: UTF8.self)) {
                pairs.append(pair)
            }
        }
        return pairs
    }

    public static func load(directory: String) -> [String: String] {
        var values: [String: String] = [:]
        for file in files {
            values.merge(read(file: (directory as NSString).appendingPathComponent(file))) { _, new in new }
        }
        return values
    }

    static func read(file path: String) -> [String: String] {
        guard let data = FileManager.default.contents(atPath: path) else {
            return [:]
        }
        var values: [String: String] = [:]
        for pair in parse(String(decoding: data, as: UTF8.self)) {
            values[pair.key] = pair.value
        }
        return values
    }

    private static func pair(from rawLine: String) -> (key: String, value: String)? {
        var line = rawLine.trimmingCharacters(in: .whitespacesAndNewlines)
        if line.isEmpty || line.hasPrefix("#") {
            return nil
        }
        if line.hasPrefix("export ") {
            line = String(line.dropFirst("export ".count))
        }
        guard let separator = line.firstIndex(of: "=") else {
            return nil
        }
        let key = line[..<separator].trimmingCharacters(in: .whitespacesAndNewlines)
        let value = line[line.index(after: separator)...]
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .trimmingCharacters(in: CharacterSet(charactersIn: "\"'"))
        return key.isEmpty ? nil : (key: key, value: value)
    }
}
