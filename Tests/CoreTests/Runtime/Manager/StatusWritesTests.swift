import Testing

@testable import Core

struct StatusWritesTests {
    @Test func aStaleWriteCannotOverwriteANewerOne() async throws {
        let store = try Store.inMemory()
        try await store.upsert(Project(id: "p", name: "p", path: "/tmp/p", slug: "p", localDomain: "p.orbit.test"))
        let writes = StatusWrites()
        let running = writes.ticket(for: "p")
        let failed = writes.ticket(for: "p")

        await writes.write(.error, projectID: "p", ticket: failed, to: store)
        await writes.write(.running, projectID: "p", ticket: running, to: store)

        #expect(try await store.project(id: "p")?.status == .error)
    }

    @Test func ticketsAreCountedPerProject() async throws {
        let store = try Store.inMemory()
        for id in ["a", "b"] {
            try await store.upsert(
                Project(id: id, name: id, path: "/tmp/\(id)", slug: id, localDomain: "\(id).orbit.test"))
        }
        let writes = StatusWrites()
        let first = writes.ticket(for: "a")
        let other = writes.ticket(for: "b")

        await writes.write(.stopped, projectID: "b", ticket: other, to: store)
        await writes.write(.running, projectID: "a", ticket: first, to: store)

        #expect(try await store.project(id: "a")?.status == .running)
        #expect(try await store.project(id: "b")?.status == .stopped)
    }
}
