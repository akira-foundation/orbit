import Foundation

extension Store {
    private static let incrementalVacuumMode: Int64 = 2
    private static let adoptionFailurePrefix = "auto vacuum adoption failed: "

    static func adoptIncrementalVacuum(
        _ db: SQLiteDatabase,
        vacuum: (SQLiteDatabase) throws -> Void = { try $0.execute("PRAGMA auto_vacuum = INCREMENTAL; VACUUM;") }
    ) throws {
        let mode = try db.query("PRAGMA auto_vacuum;").first?.int("auto_vacuum")
        guard mode != incrementalVacuumMode, try !hasRecordedAdoptionFailure(db) else { return }
        do {
            try vacuum(db)
        } catch {
            try? insertEvents(
                [
                    EventRecord(
                        timestamp: Int64(Date.now.timeIntervalSince1970 * 1_000_000_000), level: .warn, source: .system,
                        text: adoptionFailurePrefix + "\(error)")
                ], into: db)
        }
    }

    private static func hasRecordedAdoptionFailure(_ db: SQLiteDatabase) throws -> Bool {
        try !db.query(
            "SELECT id FROM events WHERE project_id IS NULL AND text LIKE ? LIMIT 1;",
            [.text(adoptionFailurePrefix + "%")]
        ).isEmpty
    }
}
