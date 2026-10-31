import Foundation
import SQLite3

private let sqliteTransient = unsafeBitCast(-1, to: sqlite3_destructor_type.self)

final class SQLiteDatabase: @unchecked Sendable {
    private var handle: OpaquePointer?

    init(path: String) throws {
        var handle: OpaquePointer?
        let flags = SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE | SQLITE_OPEN_FULLMUTEX
        guard sqlite3_open_v2(path, &handle, flags, nil) == SQLITE_OK, let handle else {
            let message = handle.map { String(cString: sqlite3_errmsg($0)) } ?? "could not open \(path)"
            sqlite3_close_v2(handle)
            throw SQLiteError(message: message, sql: nil)
        }
        self.handle = handle
        try execute("PRAGMA foreign_keys = ON;")
        try execute("PRAGMA busy_timeout = 5000;")
        if path != ":memory:" {
            try enableWriteAheadLogging()
        }
    }

    private func enableWriteAheadLogging() throws {
        let deadline = ContinuousClock.now + .seconds(5)
        while true {
            do {
                if try query("PRAGMA journal_mode;").first?.string("journal_mode") == "wal" { return }
                try execute("PRAGMA journal_mode = WAL;")
                return
            } catch let error as SQLiteError where error.code == SQLITE_BUSY && ContinuousClock.now < deadline {
                Thread.sleep(forTimeInterval: 0.005)
            }
        }
    }

    deinit {
        sqlite3_close_v2(handle)
    }

    private func fail(_ sql: String?) -> SQLiteError {
        let message = handle.map { String(cString: sqlite3_errmsg($0)) } ?? "database is closed"
        return SQLiteError(message: message, sql: sql, code: handle.map { sqlite3_errcode($0) & 0xFF } ?? 0)
    }

    func execute(_ sql: String) throws {
        guard sqlite3_exec(handle, sql, nil, nil, nil) == SQLITE_OK else { throw fail(sql) }
    }

    private func prepare(_ sql: String, _ bindings: [SQLValue]) throws -> OpaquePointer {
        var statement: OpaquePointer?
        guard sqlite3_prepare_v2(handle, sql, -1, &statement, nil) == SQLITE_OK, let statement else {
            throw fail(sql)
        }
        for (offset, value) in bindings.enumerated() {
            let index = Int32(offset + 1)
            let status: Int32 =
                switch value {
                case .null: sqlite3_bind_null(statement, index)
                case .int(let number): sqlite3_bind_int64(statement, index, number)
                case .double(let number): sqlite3_bind_double(statement, index, number)
                case .text(let text):
                    text.withCString {
                        sqlite3_bind_text64(
                            statement, index, $0, UInt64(text.utf8.count), sqliteTransient, UInt8(SQLITE_UTF8))
                    }
                }
            guard status == SQLITE_OK else {
                sqlite3_finalize(statement)
                throw fail(sql)
            }
        }
        return statement
    }

    private func value(of statement: OpaquePointer, at index: Int32) -> SQLValue {
        switch sqlite3_column_type(statement, index) {
        case SQLITE_INTEGER: .int(sqlite3_column_int64(statement, index))
        case SQLITE_FLOAT: .double(sqlite3_column_double(statement, index))
        case SQLITE_TEXT:
            .text(
                String(
                    decoding: UnsafeBufferPointer(
                        start: sqlite3_column_text(statement, index),
                        count: Int(sqlite3_column_bytes(statement, index))),
                    as: UTF8.self))
        default: .null
        }
    }

    func query(_ sql: String, _ bindings: [SQLValue] = []) throws -> [SQLiteRow] {
        let statement = try prepare(sql, bindings)
        defer { sqlite3_finalize(statement) }

        let columnCount = sqlite3_column_count(statement)
        let names = (0..<columnCount).map { String(cString: sqlite3_column_name(statement, $0)) }

        var rows: [SQLiteRow] = []
        while true {
            let step = sqlite3_step(statement)
            if step == SQLITE_DONE { break }
            guard step == SQLITE_ROW else { throw fail(sql) }
            var columns: [String: SQLValue] = [:]
            for index in 0..<columnCount {
                columns[names[Int(index)]] = value(of: statement, at: index)
            }
            rows.append(SQLiteRow(columns: columns))
        }
        return rows
    }

    @discardableResult
    func run(_ sql: String, _ bindings: [SQLValue] = []) throws -> Int {
        let statement = try prepare(sql, bindings)
        defer { sqlite3_finalize(statement) }
        let step = sqlite3_step(statement)
        guard step == SQLITE_DONE || step == SQLITE_ROW else { throw fail(sql) }
        return Int(sqlite3_changes(handle))
    }

    func transaction<T>(_ body: () throws -> T) throws -> T {
        try within("BEGIN IMMEDIATE;", body)
    }

    func read<T>(_ body: () throws -> T) throws -> T {
        try within("BEGIN DEFERRED;", body)
    }

    private func within<T>(_ begin: String, _ body: () throws -> T) throws -> T {
        try execute(begin)
        do {
            let result = try body()
            try execute("COMMIT;")
            return result
        } catch {
            try? execute("ROLLBACK;")
            throw error
        }
    }

    var userVersion: Int32 {
        get throws {
            guard let value = try query("PRAGMA user_version;").first?.int("user_version"),
                let version = Int32(exactly: value)
            else {
                throw SQLiteError(message: "Could not read the database schema version", sql: nil)
            }
            return version
        }
    }
}
