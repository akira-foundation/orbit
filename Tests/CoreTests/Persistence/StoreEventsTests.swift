import Foundation
import Testing

@testable import Core

struct StoreEventsTests {
    private func store(with names: String...) async throws -> (Store, [String]) {
        let store = try Store.inMemory()
        var ids: [String] = []
        for name in names {
            let analysis = ProjectAnalysis(name: name, path: "/tmp/\(name)", runtimeKind: .phpFPM, phpVersion: "8.3")
            ids.append(
                try await store.upsert(Project(analysis: analysis, slug: name, localDomain: "\(name).orbit.test")).id)
        }
        return (store, ids)
    }

    private func record(_ projectID: String?, _ text: String, at timestamp: Int64 = 1) -> EventRecord {
        EventRecord(
            timestamp: timestamp, projectID: projectID, sessionID: "s", level: .info, source: .runtime,
            stream: .stdout, text: text)
    }

    @Test func storeKeepsEventsWithoutProject() async throws {
        let (store, ids) = try await store(with: "p")
        try await store.appendEvents([record(nil, "app-wide"), record("", "blank"), record(ids[0], "mine")])

        let lines = try await store.events(projectID: ids[0], since: 0, limit: 0)
        #expect(lines.map(\.text) == ["mine"])
        let total = try store.db.query("SELECT COUNT(*) AS n FROM events WHERE project_id IS NULL;")
        #expect(total.first?.int("n") == 2)
    }

    @Test func rejectedRowDoesNotAbortTheBatch() async throws {
        let (store, ids) = try await store(with: "p")
        try await store.appendEvents([record(ids[0], "before"), record("missing", "orphan"), record(ids[0], "after")])
        #expect(try await store.events(projectID: ids[0], since: 0, limit: 10).map(\.text) == ["before", "after"])
    }

    @Test func historyClampsTheLimit() async throws {
        let (store, ids) = try await store(with: "p")
        try await store.appendEvents((0..<1200).map { record(ids[0], "\($0)", at: Int64($0)) })

        #expect(try await store.events(projectID: ids[0], since: 0, limit: 0).count == 1000)
        #expect(try await store.events(projectID: ids[0], since: 0, limit: -1).count == 1000)
        #expect(try await store.events(projectID: ids[0], since: 0, limit: 6000).count == 1000)
        #expect(try await store.events(projectID: ids[0], since: 0, limit: 5000).count == 1200)
        #expect(try await store.events(projectID: ids[0], since: 0, limit: 5).count == 5)
        #expect(try await store.events(projectID: ids[0], since: 1195, limit: 0).count == 5)
    }

    @Test func historyOrdersByTimestampThenInsertion() async throws {
        let (store, ids) = try await store(with: "p")
        try await store.appendEvents([record(ids[0], "late", at: 20), record(ids[0], "first", at: 10)])
        try await store.appendEvents([record(ids[0], "second", at: 10)])

        let lines = try await store.events(projectID: ids[0], since: 0, limit: 10)
        #expect(lines.map(\.text) == ["first", "second", "late"])
        #expect(lines[0].timestamp == Date(timeIntervalSince1970: 10e-9))
    }

    @Test func systemRowsComeBackWithTheSystemStream() async throws {
        let (store, ids) = try await store(with: "p")
        let marker = EventRecord(timestamp: 5, projectID: ids[0], level: .warn, source: .system, text: "stopped")
        try await store.appendEvents([marker, record(ids[0], "out", at: 6)])

        let lines = try await store.events(projectID: ids[0], since: 0, limit: 10)
        #expect(lines.map(\.stream) == [.system, .stdout])
        let row = try store.db.query("SELECT level, source, session_id FROM events WHERE ts = 5;").first
        #expect(row?.string("level") == "warn")
        #expect(row?.string("source") == "system")
        #expect(row?.string("session_id") == "")
    }

    @Test func deletingAProjectCascadesItsEvents() async throws {
        let (store, ids) = try await store(with: "p", "q")
        try await store.appendEvents([record(ids[0], "gone"), record(ids[1], "kept"), record(nil, "global")])

        try await store.deleteProject(id: ids[0])

        let rows = try store.db.query("SELECT text FROM events ORDER BY id;")
        #expect(rows.compactMap { $0.string("text") } == ["kept", "global"])
    }

    @Test func versionOneDatabaseGainsTheEventsTable() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let legacy = try SQLiteDatabase(path: path)
        try legacy.execute(StoreMigrations.all[0])
        try legacy.execute("PRAGMA user_version = 1;")

        let store = try Store(path: path)
        #expect(try await store.schemaVersion() == 2)
        try await store.appendEvents([record(nil, "after upgrade")])
        let indexes = try store.db.query(
            "SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'events' ORDER BY name;")
        #expect(
            indexes.compactMap { $0.string("name") } == [
                "idx_events_level", "idx_events_project_ts", "idx_events_session", "idx_events_source",
            ])
    }
}
