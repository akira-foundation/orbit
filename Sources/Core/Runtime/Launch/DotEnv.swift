import Foundation

public enum DotEnv {
    public static let files = [".env", ".env.local", ".env.development", ".env.development.local"]

    private static let maximumLineLength = 64 * 1024

    public static func parse(_ text: String) -> [(key: String, value: String)] {
        var pairs: [(key: String, value: String)] = []
        for substring in text.split(separator: "\n", omittingEmptySubsequences: false) {
            if substring.utf8.count >= maximumLineLength {
                break
            }
            var line = substring.trimmingCharacters(in: .whitespacesAndNewlines)
            if line.isEmpty || line.hasPrefix("#") {
                continue
            }
            if line.hasPrefix("export ") {
                line = String(line.dropFirst("export ".count))
            }
            guard let separator = line.firstIndex(of: "=") else {
                continue
            }
            let key = line[..<separator].trimmingCharacters(in: .whitespacesAndNewlines)
            let value = line[line.index(after: separator)...]
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .trimmingCharacters(in: CharacterSet(charactersIn: "\"'"))
            if !key.isEmpty {
                pairs.append((key: key, value: value))
            }
        }
        return pairs
    }

    public static func load(directory: String) -> [String: String] {
        var values: [String: String] = [:]
        for file in files {
            let path = (directory as NSString).appendingPathComponent(file)
            guard let data = FileManager.default.contents(atPath: path) else {
                continue
            }
            for pair in parse(String(decoding: data, as: UTF8.self)) {
                values[pair.key] = pair.value
            }
        }
        return values
    }
}
