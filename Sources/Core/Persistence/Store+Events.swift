import Foundation

extension Store {
    static let historyByteBudget = 16 * 1024 * 1024
    static let historyFetchRows = 32

    public func appendEvents(_ records: [EventRecord]) throws {
        try Self.insertEvents(records, into: db)
    }

    static func insertEvents(_ records: [EventRecord], into db: SQLiteDatabase) throws {
        try db.transaction {
            for record in records {
                let projectID = record.projectID.flatMap { $0.isEmpty ? nil : $0 }
                do {
                    try db.run(
                        """
                        INSERT INTO events (project_id, session_id, ts, stream, text, level, source)
                        VALUES (?, ?, ?, ?, ?, ?, ?);
                        """,
                        [
                            .optional(projectID), .text(record.sessionID ?? ""), .int(record.timestamp),
                            .text(record.stream?.rawValue ?? ""), .text(record.text), .text(record.level.rawValue),
                            .text(record.source.rawValue),
                        ])
                } catch let error as SQLiteError where error.isConstraintViolation {
                    continue
                }
            }
        }
    }

    public func events(projectID: String, since: Int64, limit: Int) throws -> [LogLine] {
        try eventPage(projectID: projectID, after: EventCursor(timestamp: since, id: 0), limit: limit)
    }

    public func events(projectID: String, after cursor: EventCursor, limit: Int) throws -> [LogLine] {
        try eventPage(projectID: projectID, after: cursor, limit: limit)
    }

    private func eventPage(projectID: String, after start: EventCursor, limit: Int) throws -> [LogLine] {
        let limit = limit <= 0 || limit > 5000 ? 1000 : limit
        var cursor = start
        var lines: [LogLine] = []
        var bytes = 0
        while lines.count < limit {
            let rows = try db.query(
                """
                SELECT id, ts, stream, text FROM events
                WHERE project_id = ? AND (ts, id) > (?, ?) ORDER BY ts ASC, id ASC LIMIT ?;
                """,
                [
                    .text(projectID), .int(cursor.timestamp), .int(cursor.id),
                    .int(Int64(min(Self.historyFetchRows, limit - lines.count))),
                ])
            guard !rows.isEmpty else { break }
            for row in rows {
                let line = try Self.logLine(row)
                lines.append(line)
                bytes += line.text.utf8.count
                cursor = line.cursor ?? cursor
                if bytes >= Self.historyByteBudget { return lines }
            }
        }
        return lines
    }

    private static func logLine(_ row: SQLiteRow) throws -> LogLine {
        let nanoseconds = try row.requiredInt("ts", in: "events")
        return LogLine(
            timestamp: Date(timeIntervalSince1970: Double(nanoseconds) / 1_000_000_000),
            stream: row.string("stream").flatMap(LogStream.init(rawValue:)) ?? .system,
            text: try row.requiredText("text", in: "events"),
            cursor: EventCursor(timestamp: nanoseconds, id: try row.requiredInt("id", in: "events")))
    }
}
