import Foundation
import Testing

@testable import Core

struct EventLogTests {
    private func store() async throws -> (Store, String) {
        let store = try Store.inMemory()
        let analysis = ProjectAnalysis(name: "p", path: "/tmp/p", runtimeKind: .phpFPM, phpVersion: "8.3")
        let project = try await store.upsert(Project(analysis: analysis, slug: "p", localDomain: "p.orbit.test"))
        return (store, project.id)
    }

    private func record(_ projectID: String, _ text: String) -> EventRecord {
        EventRecord(projectID: projectID, source: .runtime, stream: .stdout, text: text)
    }

    @Test func eventLogFlushesAtBatchSizeWithoutWaitingForTheTimer() async throws {
        let (store, projectID) = try await store()
        let log = EventLog(store: store, clock: TestClock())
        for index in 0..<EventLog.batchSize {
            log.push(record(projectID, "\(index)"))
        }

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
