import Foundation
import Testing

@testable import Core

struct StoreEventPaginationTests {
    private static let walkLimit = 100_000

    private func store() async throws -> (Store, String) {
        let store = try Store.inMemory()
        let analysis = ProjectAnalysis(name: "p", path: "/tmp/p", runtimeKind: .phpFPM, phpVersion: "8.3")
        let project = try await store.upsert(Project(analysis: analysis, slug: "p", localDomain: "p.orbit.test"))
        return (store, project.id)
    }

    private func walk(_ store: Store, _ projectID: String, pageSize: Int) async throws -> [String] {
        var seen: [String] = []
        var page = try await store.events(projectID: projectID, since: 0, limit: pageSize)
        while let last = page.last, seen.count < Self.walkLimit {
            seen += page.map(\.text)
            let cursor = try #require(last.cursor)
            page = try await store.events(projectID: projectID, after: cursor, limit: pageSize)
        }
        return seen
    }

    @Test func cursorNeitherLosesNorRepeatsLinesThatShareATimestamp() async throws {
        let (store, projectID) = try await store()
        let records = (0..<1500).map { EventRecord(timestamp: 1000, projectID: projectID, text: "\($0)") }
        try await store.appendEvents(records)

        let seen = try await walk(store, projectID, pageSize: 100)

        #expect(seen == records.map(\.text))
    }

    @Test func cursorKeepsNanosecondsThatADoubleDateWouldRound() async throws {
        let (store, projectID) = try await store()
        let base: Int64 = 1_800_000_000_000_000_000
        let records = (0..<200).map { EventRecord(timestamp: base + Int64($0), projectID: projectID, text: "\($0)") }
        try await store.appendEvents(records)

        let seen = try await walk(store, projectID, pageSize: 7)

        #expect(seen == records.map(\.text))
    }

    @Test func cursorFollowsTimestampOrderAcrossInsertionOrder() async throws {
        let (store, projectID) = try await store()
        try await store.appendEvents([
            EventRecord(timestamp: 20, projectID: projectID, text: "late"),
            EventRecord(timestamp: 10, projectID: projectID, text: "first"),
            EventRecord(timestamp: 10, projectID: projectID, text: "second"),
        ])

        #expect(try await walk(store, projectID, pageSize: 1) == ["first", "second", "late"])
    }

    @Test func aPageStopsAtTheByteBudgetAndTheCursorReachesTheRest() async throws {
        let (store, projectID) = try await store()
        let line = String(repeating: "x", count: StoredText.maximumBytes)
        let total = Store.historyByteBudget / StoredText.maximumBytes + 40
        try await store.appendEvents(
            (0..<total).map { EventRecord(timestamp: Int64($0), projectID: projectID, text: "\(line)") })

        let first = try await store.events(projectID: projectID, since: 0, limit: 5000)
        #expect(first.count < total)
        #expect(first.reduce(0) { $0 + $1.text.utf8.count } <= Store.historyByteBudget + StoredText.maximumBytes)

        let cursor = try #require(first.last?.cursor)
        #expect(try await store.events(projectID: projectID, after: cursor, limit: 5000).count == total - first.count)
    }

    @Test func aRowLargerThanTheBudgetStillMakesProgress() async throws {
        let (store, projectID) = try await store()
        let huge = String(repeating: "x", count: Store.historyByteBudget + 1)
        try await store.appendEvents([
            EventRecord(timestamp: 1, projectID: projectID, text: huge),
            EventRecord(timestamp: 2, projectID: projectID, text: "next"),
        ])

        let first = try await store.events(projectID: projectID, since: 0, limit: 5000)
        #expect(first.count == 1)
        let cursor = try #require(first.last?.cursor)
        #expect(try await store.events(projectID: projectID, after: cursor, limit: 5000).map(\.text) == ["next"])
    }
}
