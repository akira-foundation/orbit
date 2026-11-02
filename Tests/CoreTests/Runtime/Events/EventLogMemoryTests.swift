import Foundation
import Testing

@testable import Core

struct EventLogMemoryTests {
    private func store(at path: String = ":memory:") async throws -> (Store, String) {
        let store = try Store(path: path)
        let analysis = ProjectAnalysis(name: "p", path: "/tmp/p", runtimeKind: .phpFPM, phpVersion: "8.3")
        let project = try await store.upsert(Project(analysis: analysis, slug: "p", localDomain: "p.orbit.test"))
        return (store, project.id)
    }

    private func record(_ projectID: String, _ text: String, at timestamp: Int64 = 0) -> EventRecord {
        EventRecord(timestamp: timestamp, projectID: projectID, source: .runtime, stream: .stdout, text: text)
    }

    @Test func storedTextIsCutAtALimitWithAMarkerOnAScalarBoundary() {
        #expect(StoredText.limited("short") == "short")
        let exact = String(repeating: "a", count: StoredText.maximumBytes)
        #expect(StoredText.limited(exact) == exact)
        let cut = StoredText.limited(String(repeating: "é", count: StoredText.maximumBytes))
        #expect(cut.utf8.count <= StoredText.maximumBytes)
        #expect(cut.hasSuffix(StoredText.marker))
        #expect(!cut.unicodeScalars.contains("\u{FFFD}"))
    }

    @Test func aOneMebibyteLineIsStoredCut() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: TestClock())
        log.push(record(projectID, String(repeating: "x", count: 1 << 20)))
        await log.close()

        let line = try #require(try await store.events(projectID: projectID, since: 0, limit: 0).first)
        #expect(line.text.utf8.count == StoredText.maximumBytes)
        #expect(line.text.hasSuffix(StoredText.marker))
    }

    @Test func theBufferIsLimitedByBytes() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let (store, projectID) = try await store(at: path)
        let log = EventLog(store: store, clock: TestClock())
        let blocker = try SQLiteDatabase(path: path)
        try blocker.execute("BEGIN IMMEDIATE;")
        let line = String(repeating: "x", count: 1 << 20)
        for _ in 0..<EventLog.bufferSize {
            log.push(record(projectID, line))
        }
        try blocker.execute("COMMIT;")
        await log.close()

        let rows = try store.db.query("SELECT COUNT(*) AS n FROM events;").first?.int("n") ?? 0
        let bound = Int64(EventLog.bufferBytes / StoredText.maximumBytes + EventLog.batchSize)
        #expect(rows > 0)
        #expect(rows <= bound)
    }

    @Test func timestampsNeverGoBackwardsInInsertionOrder() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: TestClock())
        log.push(record(projectID, "later", at: 2000))
        log.push(record(projectID, "stamped earlier on another thread", at: 1000))
        await log.close()

        let lines = try await store.events(projectID: projectID, since: 0, limit: 0)
        #expect(lines.map(\.text) == ["later", "stamped earlier on another thread"])
        #expect(lines.map(\.cursor?.timestamp) == [2000, 2000])
    }

    @Test func theTimerWritesWithoutAFlushOrAFullBatch() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: ManualClock())
        log.push(record(projectID, "one line"))

        #expect(
            await Poll.until {
                try await store.events(projectID: projectID, since: 0, limit: 0).map(\.text) == ["one line"]
            })
        await log.close()
    }

    @Test func expiredEventsAreDeletedWhenTheLogStarts() async throws {
        let (store, projectID) = try await store()
        let clock = TestClock()
        let old = Int64(clock.now.timeIntervalSince1970 - 30 * 86_400) * 1_000_000_000
        try await store.appendEvents([record(projectID, "old", at: old)])
        let log = EventLog(store: store, clock: clock)

        #expect(await Poll.until { try await store.eventCount(projectID: projectID) == 0 })
        await log.close()
    }

    @Test func retentionRunsAgainAfterItsInterval() async throws {
        let (store, projectID) = try await store()
        let clock = ManualClock()
        let log = EventLog(store: store, clock: clock)
        #expect(await clock.waitForPending(EventRetention.interval))
        let old = Int64(clock.now.timeIntervalSince1970 - 30 * 86_400) * 1_000_000_000
        try await store.appendEvents([record(projectID, "old", at: old)])

        clock.advance(by: Double(EventRetention.interval.components.seconds))

        #expect(await Poll.until { try await store.eventCount(projectID: projectID) == 0 })
        await log.close()
    }
}
