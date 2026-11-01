import Foundation
import Synchronization
import Testing

@testable import Core

struct EventLogTests {
    private func store(at path: String = ":memory:") async throws -> (Store, String) {
        let store = try Store(path: path)
        let analysis = ProjectAnalysis(name: "p", path: "/tmp/p", runtimeKind: .phpFPM, phpVersion: "8.3")
        let project = try await store.upsert(Project(analysis: analysis, slug: "p", localDomain: "p.orbit.test"))
        return (store, project.id)
    }

    private func record(_ projectID: String, _ text: String) -> EventRecord {
        EventRecord(projectID: projectID, source: .runtime, stream: .stdout, text: text)
    }

    private func lockWrites(_ path: String) throws -> SQLiteDatabase {
        let blocker = try SQLiteDatabase(path: path)
        try blocker.execute("BEGIN IMMEDIATE;")
        return blocker
    }

    @Test func eventLogFlushesAtBatchSizeWithoutWaitingForTheTimer() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: TestClock())
        for index in 1..<EventLog.batchSize {
            log.push(record(projectID, "\(index)"))
        }
        try await Task.sleep(for: .milliseconds(100))
        #expect(try await log.history(projectID: projectID, since: 0, limit: 0).isEmpty)

        log.push(record(projectID, "last"))
        var lines: [LogLine] = []
        let deadline = ContinuousClock.now + .seconds(5)
        while lines.count < EventLog.batchSize && ContinuousClock.now < deadline {
            try await Task.sleep(for: .milliseconds(5))
            lines = try await log.history(projectID: projectID, since: 0, limit: 0)
        }
        #expect(lines.count == EventLog.batchSize)
        await log.close()
    }

    @Test func flushWritesWhatWasPushedBeforeIt() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: TestClock())
        log.push(record(projectID, "one"))
        log.push(record(projectID, "two"))

        await log.flush()

        #expect(try await log.history(projectID: projectID, since: 0, limit: 0).map(\.text) == ["one", "two"])
        await log.close()
    }

    @Test func flushWaitsForAWriteAlreadyInFlight() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let (store, projectID) = try await store(at: path)
        let log = EventLog(store: store, clock: TestClock())
        let blocker = try lockWrites(path)
        for index in 0..<EventLog.batchSize {
            log.push(record(projectID, "\(index)"))
        }
        try await Task.sleep(for: .milliseconds(50))

        let flushed = Atomic<Bool>(false)
        let flushing = Task {
            await log.flush()
            flushed.store(true, ordering: .relaxed)
        }
        try await Task.sleep(for: .milliseconds(100))
        #expect(flushed.load(ordering: .relaxed) == false)

        try blocker.execute("COMMIT;")
        await flushing.value
        #expect(try await store.events(projectID: projectID, since: 0, limit: 0).count == EventLog.batchSize)
        await log.close()
    }

    @Test func overflowDropsWhatTheBufferCannotHold() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let (store, projectID) = try await store(at: path)
        let log = EventLog(store: store, clock: TestClock())
        let blocker = try lockWrites(path)
        for index in 0..<EventLog.batchSize {
            log.push(record(projectID, "in flight \(index)"))
        }
        let deadline = ContinuousClock.now + .seconds(5)
        while await log.received < EventLog.batchSize && ContinuousClock.now < deadline {
            try await Task.sleep(for: .milliseconds(1))
        }
        #expect(await log.received == EventLog.batchSize)
        for index in 0..<(EventLog.bufferSize + 50) {
            log.push(record(projectID, "\(index)"))
        }

        try blocker.execute("COMMIT;")
        await log.close()

        let rows = try store.db.query("SELECT COUNT(*) AS n FROM events;")
        #expect(rows.first?.int("n") == Int64(EventLog.batchSize + EventLog.bufferSize))
    }

    @Test func failedWriteIsKeptAsTheLastWriteError() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: TestClock())
        #expect(await log.lastWriteError == nil)
        try store.db.execute("DROP TABLE events;")

        log.push(record(projectID, "lost"))
        await log.flush()

        #expect(await log.lastWriteError != nil)
        await log.close()
    }

    @Test func pushFillsTheTimestampFromTheClock() async throws {
        let (store, projectID) = try await store()
        let clock = TestClock()
        let log = EventLog(store: store, clock: clock)
        log.push(record(projectID, "stamped"))
        await log.close()

        let line = try #require(try await store.events(projectID: projectID, since: 0, limit: 0).first)
        #expect(abs(line.timestamp.timeIntervalSince(clock.now)) < 0.001)
    }

    @Test func eventLogDrainsOnClose() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: TestClock())
        log.push(record(projectID, "a"))
        log.push(record(projectID, "b"))
        log.push(record(projectID, "c"))

        await log.close()

        #expect(try await store.events(projectID: projectID, since: 0, limit: 0).map(\.text) == ["a", "b", "c"])
        log.push(record(projectID, "late"))
        await log.flush()
        #expect(try await store.events(projectID: projectID, since: 0, limit: 0).count == 3)
    }

    @Test func eventLogWithoutStoreIsANoop() async throws {
        let log = EventLog(store: nil, clock: TestClock())
        log.push(EventRecord(text: "nowhere"))
        await log.flush()
        #expect(try await log.history(projectID: "p", since: 0, limit: 0).isEmpty)
        await log.close()
    }

    @Test func recordingSinkKeepsEmissionOrder() {
        let sink = RecordingEventSink()
        let line = LogLine(timestamp: .now, stream: .stderr, text: "oops")
        let snapshot = RuntimeSession(projectID: "p", sessionID: "s", clock: TestClock()).snapshot(conns: 0)

        sink.emit(.log(projectID: "p", line))
        sink.emit(.status(.starting, snapshot))

        #expect(sink.events == [.log(projectID: "p", line), .status(.starting, snapshot)])
        #expect(RuntimeEventName.crash.rawValue == "runtime:crash")
    }
}
