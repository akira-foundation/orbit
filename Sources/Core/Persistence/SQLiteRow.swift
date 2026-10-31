struct SQLiteRow: Sendable {
    let columns: [String: SQLValue]

    subscript(key: String) -> SQLValue { columns[key] ?? .null }

    func string(_ key: String) -> String? { self[key].stringValue }
    func int(_ key: String) -> Int64? { self[key].intValue }
    func double(_ key: String) -> Double? { self[key].doubleValue }
    func bool(_ key: String) -> Bool { (self[key].intValue ?? 0) != 0 }
}
