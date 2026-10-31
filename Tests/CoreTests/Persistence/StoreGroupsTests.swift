import Foundation
import Testing

@testable import Core

@Suite struct StoreGroupsTests {
    private func project(_ name: String) -> Project {
        let analysis = ProjectAnalysis(name: name, path: "/tmp/\(name)", runtimeKind: .phpFPM, phpVersion: "8.3")
        return Project(analysis: analysis, slug: name, localDomain: "\(name).orbit.test")
    }

    @Test func groupsCRUD() async throws {
        let store = try Store.inMemory()
        let group = try await store.createGroup(named: "stack")
        let api = try await store.upsert(project("api"))

        try await store.addProject(api.id, toGroup: group.id)
        let listed = try await store.groups()
        #expect(group.name == "stack")
        #expect(!group.id.isEmpty)
        #expect(listed.count == 1)
        #expect(listed[0].projectIDs == [api.id])

        try await store.removeProject(api.id, fromGroup: group.id)
        #expect(try await store.groups()[0].projectIDs.isEmpty)

        try await store.deleteGroup(id: group.id)
        #expect(try await store.groups().isEmpty)
    }

    @Test func addingTwiceKeepsOneMembership() async throws {
        let store = try Store.inMemory()
        let group = try await store.createGroup(named: "stack")
        let api = try await store.upsert(project("api"))

        try await store.addProject(api.id, toGroup: group.id)
        try await store.addProject(api.id, toGroup: group.id)

        #expect(try await store.groups()[0].projectIDs == [api.id])
    }

    @Test func deletingAProjectRemovesItFromGroups() async throws {
        let store = try Store.inMemory()
        let group = try await store.createGroup(named: "stack")
        let api = try await store.upsert(project("api"))
        try await store.addProject(api.id, toGroup: group.id)

        try await store.deleteProject(id: api.id)

        #expect(try await store.groups()[0].projectIDs.isEmpty)
    }

    @Test func newGroupsGoLast() async throws {
        let store = try Store.inMemory()
        let first = try await store.createGroup(named: "a")
        let second = try await store.createGroup(named: "b")

        #expect(first.sortOrder == 0)
        #expect(second.sortOrder == 1)
        #expect(try await store.groups().map(\.id) == [first.id, second.id])
    }

    @Test func membersAreOrderedByProjectName() async throws {
        let store = try Store.inMemory()
        let group = try await store.createGroup(named: "stack")
        let zeta = try await store.upsert(project("zeta"))
        let alpha = try await store.upsert(project("Alpha"))
        try await store.addProject(zeta.id, toGroup: group.id)
        try await store.addProject(alpha.id, toGroup: group.id)

        #expect(try await store.groups()[0].projectIDs == [alpha.id, zeta.id])
    }

    @Test func updateChangesNameAndOrder() async throws {
        let store = try Store.inMemory()
        let first = try await store.createGroup(named: "a")
        let second = try await store.createGroup(named: "b")

        let updated = try await store.update(groupID: second.id) {
            $0.name = "renamed"
            $0.sortOrder = -1
        }

        let listed = try await store.groups()
        #expect(updated.name == "renamed")
        #expect(listed.map(\.id) == [second.id, first.id])
        #expect(listed[0].name == "renamed")
        #expect(abs(listed[0].createdAt.timeIntervalSince(second.createdAt)) < 0.001)
    }

    @Test func updatingAMissingGroupThrows() async throws {
        let store = try Store.inMemory()

        await #expect(throws: StoreError.groupNotFound("missing")) {
            try await store.update(groupID: "missing") { $0.name = "x" }
        }
    }

    @Test func addingToAMissingGroupThrows() async throws {
        let store = try Store.inMemory()
        let api = try await store.upsert(project("api"))

        await #expect(throws: StoreError.groupNotFound("missing")) {
            try await store.addProject(api.id, toGroup: "missing")
        }
    }

    @Test func addingAMissingProjectThrows() async throws {
        let store = try Store.inMemory()
        let group = try await store.createGroup(named: "stack")

        await #expect(throws: StoreError.projectNotFound("missing")) {
            try await store.addProject("missing", toGroup: group.id)
        }
    }

    @Test func deletingAMissingGroupThrows() async throws {
        let store = try Store.inMemory()

        await #expect(throws: StoreError.groupNotFound("missing")) {
            try await store.deleteGroup(id: "missing")
        }
    }
}
