struct SQLiteRow: Sendable {
    let columns: [String: SQLValue]

    subscript(key: String) -> SQLValue { columns[key] ?? .null }

    func string(_ key: String) -> String? { self[key].stringValue }
    func int(_ key: String) -> Int64? { self[key].intValue }
    func double(_ key: String) -> Double? { self[key].doubleValue }
    func bool(_ key: String) -> Bool { (self[key].intValue ?? 0) != 0 }
}

extension SQLiteRow {
    func requiredText(_ key: String, in table: String) throws -> String {
        guard case .text(let value) = self[key] else {
            throw StoreError.corruptRow(table: table, column: key, value: string(key))
        }
        return value
    }

    func requiredDouble(_ key: String, in table: String) throws -> Double {
        guard let value = double(key) else {
            throw StoreError.corruptRow(table: table, column: key, value: string(key))
        }
        return value
    }

    func requiredRaw<T: RawRepresentable>(_ key: String, in table: String) throws -> T where T.RawValue == String {
        guard let value = string(key).flatMap(T.init(rawValue:)) else {
            throw StoreError.corruptRow(table: table, column: key, value: string(key))
        }
        return value
    }
}
