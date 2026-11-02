import Foundation
import Testing

@testable import Core

struct StoreVacuumAdoptionTests {
    private func legacyDatabase(at path: String) throws -> SQLiteDatabase {
        _ = try Store(path: path)
        let db = try SQLiteDatabase(path: path)
        try db.execute("PRAGMA auto_vacuum = NONE; VACUUM;")
        return db
    }

    private func mode(_ db: SQLiteDatabase) throws -> Int64? {
        try db.query("PRAGMA auto_vacuum;").first?.int("auto_vacuum")
    }

    @Test func aFailedAdoptionIsRecordedAndNotRepeatedOnTheNextOpen() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let db = try legacyDatabase(at: path)
        var attempts = 0

        for _ in 0..<2 {
            try Store.adoptIncrementalVacuum(db) { _ in
                attempts += 1
                throw RuntimeError.message("disk full")
            }
        }

        #expect(attempts == 1)
        let store = try Store(path: path)
        let row = try store.db.query("SELECT text, level, source FROM events WHERE project_id IS NULL;")
        #expect(row.count == 1)
        #expect(row.first?.string("text") == "auto vacuum adoption failed: disk full")
        #expect(row.first?.string("level") == EventLevel.warn.rawValue)
        #expect(row.first?.string("source") == EventSource.system.rawValue)
        #expect(try mode(store.db) == 0)
    }

    @Test func adoptionIsTriedAgainOnceTheRetentionExpiresTheFailure() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let db = try legacyDatabase(at: path)
        try Store.adoptIncrementalVacuum(db) { _ in throw RuntimeError.message("disk full") }

        try await Store(path: path).pruneEvents(now: Date.now.addingTimeInterval(EventRetention.ageInSeconds + 60))

        #expect(try mode(Store(path: path).db) == 2)
    }

    @Test func aSuccessfulAdoptionRecordsNothing() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let db = try legacyDatabase(at: path)

        try Store.adoptIncrementalVacuum(db)

        #expect(try mode(db) == 2)
        #expect(try db.query("SELECT id FROM events;").isEmpty)
    }
}
