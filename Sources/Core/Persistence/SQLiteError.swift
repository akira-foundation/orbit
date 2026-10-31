struct SQLiteError: Error, CustomStringConvertible {
    let message: String
    let sql: String?

    var description: String {
        if let sql { return "\(message) [\(sql.prefix(200))]" }
        return message
    }
}
