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

    @Test func twoStoresOnOneFileSeeEachOthersWrites() async throws {
        let dir = try TemporaryDirectory()
        let path = dir.path + "/orbit.sqlite"
        let writer = try Store(path: path)
        let reader = try Store(path: path)
        let project = Project(
            id: "11111111-1111-1111-1111-111111111111", name: "app", path: "/tmp/app", slug: "app",
            localDomain: "app.orbit.test")

        try await writer.upsert(project)

        #expect(try await reader.project(id: project.id)?.name == "app")
    }

    @Test func concurrentWritersFromTwoStoresAllLand() async throws {
        let dir = try TemporaryDirectory()
        let path = dir.path + "/orbit.sqlite"
        let first = try Store(path: path)
        let second = try Store(path: path)

        @Sendable func write(_ store: Store, prefix: String) async throws {
            for index in 0..<50 {
                try await store.upsert(
                    Project(
                        id: "\(prefix)-\(index)", name: "\(prefix)\(index)", path: "/tmp/\(prefix)/\(index)",
                        slug: "\(prefix)-\(index)", localDomain: "\(prefix)-\(index).orbit.test"))
            }
        }

        async let a: Void = write(first, prefix: "a")
        async let b: Void = write(second, prefix: "b")
        _ = try await (a, b)

        #expect(try await first.projects().count == 100)
    }
}
