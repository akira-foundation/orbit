import SQLite3

struct SQLiteError: Error, CustomStringConvertible {
    private static let uniquePrefix = "UNIQUE constraint failed: "

    let message: String
    let sql: String?
    var code: Int32 = 0

    var description: String {
        if let sql { return "\(message) [\(sql.prefix(200))]" }
        return message
    }

    var duplicateColumn: String? {
        guard code == SQLITE_CONSTRAINT, message.hasPrefix(Self.uniquePrefix) else { return nil }
        return message.dropFirst(Self.uniquePrefix.count)
            .split(separator: ",")
            .map { $0.split(separator: ".").last.map(String.init) ?? "" }
            .map { $0.trimmingCharacters(in: .whitespaces) }
            .joined(separator: ",")
    }
}
