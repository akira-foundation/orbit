import Foundation

extension Store {
    public func pruneEvents(
        now: Date, maximumProjectBytes: Int64 = EventRetention.maximumProjectBytes
    ) async throws {
        let cutoff = Int64((now.timeIntervalSince1970 - EventRetention.ageInSeconds) * 1_000_000_000)
        let projects: [SQLValue] =
            [.null] + (try db.query("SELECT id FROM projects;").compactMap { $0.string("id") }.map(SQLValue.text))
        var deleted = 0
        for project in projects {
            deleted += try await deleteBatches(
                "project_id IS ? AND ts < ?", [project, .int(cutoff)])
            deleted += try await deleteBeyondBudget(project, maximumProjectBytes)
        }
        if deleted > 0 {
            try await returnFreePages()
        }
    }

    func deleteEvents(ofProject projectID: String) async throws {
        _ = try await deleteBatches("project_id IS ?", [.text(projectID)])
    }

    func returnFreePages() async throws {
        var free = try freePageCount()
        while free > 0 {
            try db.execute("PRAGMA incremental_vacuum(\(EventRetention.vacuumPages));")
            let remaining = try freePageCount()
            guard remaining < free else { return }
            free = remaining
            try await pauseBetweenBatches()
        }
    }

    private func freePageCount() throws -> Int64 {
        try db.query("PRAGMA freelist_count;").first?.int("freelist_count") ?? 0
    }

    private func deleteBeyondBudget(_ project: SQLValue, _ budget: Int64) async throws -> Int {
        var used: Int64 = 0
        var cursor = EventCursor(timestamp: .max, id: .max)
        while true {
            let rows = try db.query(
                """
                SELECT id, ts, octet_length(text) AS size FROM events
                WHERE project_id IS ? AND (ts, id) < (?, ?) ORDER BY ts DESC, id DESC LIMIT ?;
                """,
                [project, .int(cursor.timestamp), .int(cursor.id), .int(Int64(EventRetention.batchRows))])
            guard !rows.isEmpty else { return 0 }
            for row in rows {
                used += (row.int("size") ?? 0) + EventRetention.rowOverheadBytes
                cursor = EventCursor(
                    timestamp: try row.requiredInt("ts", in: "events"), id: try row.requiredInt("id", in: "events"))
                if used > budget {
                    return try await deleteBatches(
                        "project_id IS ? AND (ts, id) <= (?, ?)", [project, .int(cursor.timestamp), .int(cursor.id)])
                }
            }
            try await pauseBetweenBatches()
        }
    }

    private func deleteBatches(_ condition: String, _ bindings: [SQLValue]) async throws -> Int {
        var deleted = 0
        while true {
            let count = try db.run(
                "DELETE FROM events WHERE id IN (SELECT id FROM events WHERE \(condition) LIMIT ?);",
                bindings + [.int(Int64(EventRetention.batchRows))])
            deleted += count
            guard count == EventRetention.batchRows else { return deleted }
            try await pauseBetweenBatches()
        }
    }

    private func pauseBetweenBatches() async throws {
        try Task.checkCancellation()
        await Task.yield()
    }
}
