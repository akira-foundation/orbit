import Foundation

extension Store {
    public func appendEvents(_ records: [EventRecord]) throws {
        try db.transaction {
            for record in records {
                let projectID = record.projectID.flatMap { $0.isEmpty ? nil : $0 }
                _ = try? db.run(
                    """
                    INSERT INTO events (project_id, session_id, ts, stream, text, level, source)
                    VALUES (?, ?, ?, ?, ?, ?, ?);
                    """,
                    [
                        .optional(projectID), .text(record.sessionID ?? ""), .int(record.timestamp),
                        .text(record.stream?.rawValue ?? ""), .text(record.text), .text(record.level.rawValue),
                        .text(record.source.rawValue),
                    ])
            }
        }
    }

    public func events(projectID: String, since: Int64, limit: Int) throws -> [LogLine] {
        let limit = limit <= 0 || limit > 5000 ? 1000 : limit
        let rows = try db.query(
            "SELECT ts, stream, text FROM events WHERE project_id = ? AND ts >= ? ORDER BY ts ASC, id ASC LIMIT ?;",
            [.text(projectID), .int(since), .int(Int64(limit))])
        return try rows.map { row in
            let nanoseconds = try row.requiredInt("ts", in: "events")
            return LogLine(
                timestamp: Date(timeIntervalSince1970: Double(nanoseconds) / 1_000_000_000),
                stream: row.string("stream").flatMap(LogStream.init(rawValue:)) ?? .system,
                text: try row.requiredText("text", in: "events"))
        }
    }
}
