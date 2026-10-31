import Foundation
import Testing

@testable import Core

@Suite struct StoreTests {
    @Test func migratesAFreshDatabaseToTheLatestVersion() async throws {
        let store = try Store.inMemory()
        #expect(try await store.schemaVersion() == Int32(StoreMigrations.all.count))
    }

    @Test func reopeningDoesNotReapplyMigrations() async throws {
        let dir = try TemporaryDirectory()
        let path = dir.path + "/orbit.sqlite"
        _ = try Store(path: path)
        let again = try Store(path: path)
        #expect(try await again.schemaVersion() == Int32(StoreMigrations.all.count))
    }

    @Test func fileDatabaseUsesWALAndForeignKeys() throws {
        let dir = try TemporaryDirectory()
        let db = try SQLiteDatabase(path: dir.path + "/x.sqlite")
        #expect(try db.query("PRAGMA journal_mode").first?.string("journal_mode") == "wal")
        #expect(try db.query("PRAGMA foreign_keys").first?.int("foreign_keys") == 1)
    }

    @Test func databaseURLFollowsTheEnvironment() {
        let base = URL(fileURLWithPath: "/tmp/support")
        #expect(Store.databaseURL(in: base, environment: .production).path == "/tmp/support/Orbit/orbit.sqlite")
        #expect(
            Store.databaseURL(in: base, environment: .development).path == "/tmp/support/Orbit (Dev)/orbit.sqlite")
    }

    @Test func concurrentOpensOfAFreshFileAllSucceed() async throws {
        let dir = try TemporaryDirectory()
        let path = dir.path + "/orbit.sqlite"
        let versions = try await withThrowingTaskGroup(of: Int32.self) { group in
            for _ in 0..<8 {
                group.addTask { try await Store(path: path).schemaVersion() }
            }
            return try await group.reduce(into: []) { $0.append($1) }
        }
        #expect(versions == Array(repeating: Int32(StoreMigrations.all.count), count: 8))
    }

    @Test(.disabled("needs Task 6"))
    func twoStoresOnOneFileSeeEachOthersWrites() async throws {}

    @Test(.disabled("needs Task 6"))
    func concurrentWritersFromTwoStoresAllLand() async throws {}
}
