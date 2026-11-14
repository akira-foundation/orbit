import Foundation
import Synchronization
import Testing

@testable import Core

struct StoreEventRetentionTests {
    private static let now = Date(timeIntervalSince1970: 1_800_000_000)
    private static let dayNanoseconds: Int64 = 86_400 * 1_000_000_000

    private func store(at path: String = ":memory:", with names: String...) async throws -> (Store, [String]) {
        let store = try Store(path: path)
        var ids: [String] = []
        for name in names {
            let analysis = ProjectAnalysis(name: name, path: "/tmp/\(name)", runtimeKind: .phpFPM, phpVersion: "8.3")
            ids.append(
                try await store.upsert(Project(analysis: analysis, slug: name, localDomain: "\(name).orbit.test")).id)
        }
        return (store, ids)
    }

    private func nanoseconds(daysAgo days: Int64) -> Int64 {
        Int64(Self.now.timeIntervalSince1970) * 1_000_000_000 - days * Self.dayNanoseconds
    }

    private func record(_ projectID: String?, _ text: String, daysAgo days: Int64 = 0) -> EventRecord {
        EventRecord(timestamp: nanoseconds(daysAgo: days), projectID: projectID, stream: .stdout, text: text)
    }

    @Test func pruneDeletesEventsOlderThanTheMaximumAge() async throws {
        let (store, ids) = try await store(with: "p")
        try await store.appendEvents([
            record(ids[0], "old", daysAgo: 15), record(ids[0], "edge", daysAgo: 13),
            record(nil, "old global", daysAgo: 20),
            record(nil, "new global"),
        ])

        try await store.pruneEvents(now: Self.now)

        #expect(try await store.events(projectID: ids[0], since: 0, limit: 0).map(\.text) == ["edge"])
        let global = try store.db.query("SELECT text FROM events WHERE project_id IS NULL;")
        #expect(global.compactMap { $0.string("text") } == ["new global"])
    }

    @Test func pruneKeepsTheNewestEventsOfEachProjectWithinTheBudget() async throws {
        let (store, ids) = try await store(with: "p", "q")
        let line = String(repeating: "x", count: 1000)
        let perRow = Int64(1000) + EventRetention.rowOverheadBytes
        try await store.appendEvents(
            (0..<50).map {
                EventRecord(timestamp: nanoseconds(daysAgo: 0) + Int64($0), projectID: ids[0], text: "\($0)" + line)
            }
        )
        try await store.appendEvents((0..<5).map { record(ids[1], "\($0)" + line) })

        try await store.pruneEvents(now: Self.now, maximumProjectBytes: perRow * 10 + 100)

        let kept = try await store.events(projectID: ids[0], since: 0, limit: 100).map { String($0.text.prefix(2)) }
        #expect(kept == (40..<50).map { "\($0)" })
        #expect(try await store.eventCount(projectID: ids[1]) == 5)
    }

    @Test func pruneRemovesMoreRowsThanOneBatch() async throws {
        let (store, ids) = try await store(with: "p")
        let total = EventRetention.batchRows * 2 + 7
        try await store.appendEvents((0..<total).map { record(ids[0], "\($0)", daysAgo: 30) })

        try await store.pruneEvents(now: Self.now)

        #expect(try await store.eventCount(projectID: ids[0]) == 0)
    }

    @Test func pruneLetsOtherCallersRunBetweenBatches() async throws {
        let (store, ids) = try await store(with: "p")
        let total = EventRetention.batchRows * 40
        try await store.appendEvents((0..<total).map { record(ids[0], "\($0)", daysAgo: 30) })
        let projectID = ids[0]

        let observed = await withTaskGroup(of: Bool.self) { group in
            group.addTask {
                try? await store.pruneEvents(now: Self.now)
                return false
            }
            group.addTask {
                while true {
                    let remaining = (try? await store.eventCount(projectID: projectID)) ?? 0
                    if remaining > 0 && remaining < total { return true }
                    if remaining == 0 { return false }
                    await Task.yield()
                }
            }
            var sawPartial = false
            for await partial in group where partial { sawPartial = true }
            return sawPartial
        }

        #expect(observed)
    }

    @Test func upsertReplacingRemovesEventsInBatchesAndLetsOtherCallersRun() async throws {
        let (store, ids) = try await store(with: "p", "q")
        let total = EventRetention.batchRows * 40
        try await store.appendEvents((0..<total).map { record(ids[0], "\($0)") })
        try await store.appendEvents([record(ids[1], "kept")])
        let replacedID = ids[0]
        let analysis = ProjectAnalysis(name: "r", path: "/tmp/r", runtimeKind: .phpFPM, phpVersion: "8.3")
        let replacement = Project(analysis: analysis, slug: "r", localDomain: "r.orbit.test")

        let observed = await withTaskGroup(of: Bool.self) { group in
            group.addTask {
                _ = try? await store.upsert(replacement, replacing: replacedID)
                return false
            }
            group.addTask {
                while true {
                    let remaining = (try? await store.eventCount(projectID: replacedID)) ?? 0
                    if remaining > 0 && remaining < total { return true }
                    if remaining == 0 { return false }
                    await Task.yield()
                }
            }
            var sawPartial = false
            for await partial in group where partial { sawPartial = true }
            return sawPartial
        }

        #expect(observed)
        #expect(try await store.eventCount(projectID: replacedID) == 0)
        #expect(try await store.eventCount(projectID: ids[1]) == 1)
        #expect(try await store.project(id: replacedID) == nil)
        #expect(try await store.project(id: replacement.id)?.slug == "r")
    }

    @Test func deleteProjectRemovesEventsInBatchesAndLetsOtherCallersRun() async throws {
        let (store, ids) = try await store(with: "p", "q")
        let total = EventRetention.batchRows * 40
        try await store.appendEvents((0..<total).map { record(ids[0], "\($0)") })
        try await store.appendEvents([record(ids[1], "kept")])
        let projectID = ids[0]

        let observed = await withTaskGroup(of: Bool.self) { group in
            group.addTask {
                try? await store.deleteProject(id: projectID)
                return false
            }
            group.addTask {
                while true {
                    let remaining = (try? await store.eventCount(projectID: projectID)) ?? 0
                    if remaining > 0 && remaining < total { return true }
                    if remaining == 0 { return false }
                    await Task.yield()
                }
            }
            var sawPartial = false
            for await partial in group where partial { sawPartial = true }
            return sawPartial
        }

        #expect(observed)
        #expect(try await store.project(id: projectID) == nil)
        #expect(try await store.eventCount(projectID: projectID) == 0)
        #expect(try await store.eventCount(projectID: ids[1]) == 1)
    }

    @Test func newDatabaseUsesIncrementalAutoVacuum() async throws {
        let directory = try TemporaryDirectory()
        let (store, _) = try await store(at: directory.path + "/orbit.sqlite", with: "p")
        #expect(try store.db.query("PRAGMA auto_vacuum;").first?.int("auto_vacuum") == 2)
    }

    @Test func existingDatabaseWithoutAutoVacuumIsConvertedOnce() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let legacy = try SQLiteDatabase(path: path)
        try legacy.execute(StoreMigrations.all[0])
        try legacy.execute("PRAGMA user_version = 1;")
        try legacy.execute("PRAGMA auto_vacuum = NONE; VACUUM;")
        #expect(try legacy.query("PRAGMA auto_vacuum;").first?.int("auto_vacuum") == 0)

        let store = try Store(path: path)

        #expect(try store.db.query("PRAGMA auto_vacuum;").first?.int("auto_vacuum") == 2)
        #expect(try await store.schemaVersion() == 2)
    }

    @Test func pruneReturnsFreedPagesToTheFileSystem() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let (store, ids) = try await store(at: path, with: "p")
        let line = String(repeating: "x", count: 4096)
        try await store.appendEvents((0..<2000).map { _ in record(ids[0], line, daysAgo: 30) })
        let before = try store.db.query("PRAGMA page_count;").first?.int("page_count") ?? 0

        try await store.pruneEvents(now: Self.now)

        let after = try store.db.query("PRAGMA page_count;").first?.int("page_count") ?? 0
        #expect(after < before / 2)
        #expect(try store.db.query("PRAGMA freelist_count;").first?.int("freelist_count") == 0)
    }

    @Test(.timeLimit(.minutes(1)))
    func pruneFinishesOnADatabaseThatCouldNotAdoptAutoVacuum() async throws {
        let directory = try TemporaryDirectory()
        let (store, ids) = try await store(at: directory.path + "/orbit.sqlite", with: "p")
        try store.db.execute("PRAGMA auto_vacuum = NONE; VACUUM;")
        #expect(try store.db.query("PRAGMA auto_vacuum;").first?.int("auto_vacuum") == 0)
        let line = String(repeating: "x", count: 4096)
        try await store.appendEvents((0..<200).map { _ in record(ids[0], line, daysAgo: 30) })

        try await store.pruneEvents(now: Self.now)

        #expect(try await store.eventCount(projectID: ids[0]) == 0)
    }
}
