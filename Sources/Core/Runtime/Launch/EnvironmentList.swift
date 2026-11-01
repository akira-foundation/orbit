import Foundation

public struct EnvironmentList: Sendable, Equatable {
    public var entries: [String]

    public init(_ entries: [String]) {
        self.entries = entries
    }

    public static func process() -> EnvironmentList {
        EnvironmentList(ProcessInfo.processInfo.environment.map { "\($0.key)=\($0.value)" })
    }

    public func value(of key: String) -> String? {
        entries.first { $0.hasPrefix(key + "=") }.map { String($0.dropFirst(key.count + 1)) }
    }

    public mutating func remove(_ key: String) {
        entries.removeAll { $0.hasPrefix(key + "=") }
    }

    public mutating func set(_ key: String, _ value: String) {
        let entry = "\(key)=\(value)"
        if let index = entries.firstIndex(where: { $0.hasPrefix(key + "=") }) {
            entries[index] = entry
        } else {
            entries.append(entry)
        }
    }

    public mutating func addAbsent(_ values: [String: String]) {
        for key in values.keys.sorted() where value(of: key) == nil {
            entries.append("\(key)=\(values[key] ?? "")")
        }
    }

    public mutating func override(_ values: [String: String]) {
        for key in values.keys.sorted() {
            set(key, values[key] ?? "")
        }
    }
}
