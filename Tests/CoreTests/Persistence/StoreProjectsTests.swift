import Foundation
import Testing

@testable import Core

@Suite struct StoreProjectsTests {
    private func fullProject(id: String = "11111111-1111-1111-1111-111111111111") -> Project {
        Project(
            id: id,
            name: "Inertia App",
            path: "/tmp/inertia-app-\(id)",
            slug: "inertia-\(id)",
            localDomain: "inertia-\(id).orbit.test",
            framework: "laravel",
            packageManager: "bun",
            devCommand: "composer run dev",
            devPort: 8000,
            runtimeKind: .phpFPM,
            nodeVersion: "22",
            phpVersion: "8.3",
            pythonVersion: "3.12",
            status: .idle,
            secure: true,
            pinned: true,
            installedHash: "abc123",
            parkedFolder: "/tmp/parked",
            createdAt: Date(timeIntervalSince1970: 1_700_000_000.5),
            updatedAt: Date(timeIntervalSince1970: 1_700_000_100.25),
            scripts: [
                ProjectScript(name: "build", command: "vite build"), ProjectScript(name: "dev", command: "vite"),
            ],
            processes: [
                ProcessSpec(role: .backend, kind: .phpFPM, workDirectory: "/tmp/api", port: 8000, phpVersion: "8.3"),
                ProcessSpec(role: .frontend, kind: .command, command: "npm run dev"),
            ])
    }

    @Test func upsertRoundTripsEveryField() async throws {
        let store = try Store.inMemory()
        let project = fullProject()

        try await store.upsert(project)

        #expect(try await store.project(id: project.id) == project)
    }

    @Test func upsertRoundTripsNilsAndTheDefaultBackend() async throws {
        let store = try Store.inMemory()
        let analysis = ProjectAnalysis(name: "app", path: "/tmp/app", runtimeKind: .phpFPM, phpVersion: "8.3")
        let project = Project(analysis: analysis, slug: "app", localDomain: "app.orbit.test")

        try await store.upsert(project)
        var loaded = try #require(try await store.project(id: project.id))

        #expect(abs(loaded.createdAt.timeIntervalSince(project.createdAt)) < 0.001)
        loaded.createdAt = project.createdAt
        loaded.updatedAt = project.updatedAt
        #expect(loaded == project)
        #expect(loaded.framework == nil)
        #expect(loaded.parkedFolder == nil)
        #expect(loaded.processes.first?.role == .backend)
    }

    @Test func explicitProcessListKeepsItsOrder() async throws {
        let store = try Store.inMemory()
        let project = fullProject()

        try await store.upsert(project)
        let loaded = try #require(try await store.project(id: project.id))

        #expect(loaded.processes.count == 2)
        #expect(loaded.processes[1].role == .frontend)
        #expect(loaded.processes[1].command == "npm run dev")
    }

    @Test func upsertAgainReplacesScriptsAndProcesses() async throws {
        let store = try Store.inMemory()
        var project = fullProject()
        try await store.upsert(project)

        project.scripts = [ProjectScript(name: "only", command: "echo")]
        project.processes = [ProcessSpec(role: .backend, kind: .command, command: "go run .")]
        try await store.upsert(project)

        let loaded = try #require(try await store.project(id: project.id))
        #expect(loaded.scripts == [ProjectScript(name: "only", command: "echo")])
        #expect(loaded.processes.count == 1)
        #expect(try await store.projects().count == 1)
    }

    @Test func upsertReplacingRemovesTheOldProject() async throws {
        let store = try Store.inMemory()
        let old = fullProject(id: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
        try await store.upsert(old)

        var replacement = fullProject(id: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
        replacement.path = old.path
        replacement.slug = old.slug
        replacement.localDomain = old.localDomain
        try await store.upsert(replacement, replacing: old.id)

        #expect(try await store.project(id: old.id) == nil)
        #expect(try await store.projects().map(\.id) == [replacement.id])
    }

    @Test func updateChangesOneFieldAndBumpsUpdatedAt() async throws {
        let store = try Store.inMemory()
        var project = fullProject()
        project.updatedAt = .distantPast
        try await store.upsert(project)

        let updated = try await store.update(projectID: project.id) {
            $0.status = .running
            $0.secure = false
        }

        var loaded = try #require(try await store.project(id: project.id))
        #expect(abs(loaded.updatedAt.timeIntervalSince(updated.updatedAt)) < 0.001)
        loaded.updatedAt = updated.updatedAt
        #expect(loaded == updated)
        #expect(loaded.status == .running)
        #expect(loaded.secure == false)
        #expect(loaded.name == project.name)
        #expect(loaded.scripts == project.scripts)
        #expect(abs(loaded.updatedAt.timeIntervalSinceNow) < 5)
    }

    @Test func updateKeepsTheID() async throws {
        let store = try Store.inMemory()
        let project = fullProject()
        try await store.upsert(project)

        let updated = try await store.update(projectID: project.id) { $0.id = "other" }

        #expect(updated.id == project.id)
        #expect(try await store.project(id: "other") == nil)
    }

    @Test func updateOfAMissingProjectThrows() async throws {
        let store = try Store.inMemory()

        await #expect(throws: StoreError.projectNotFound("nope")) {
            try await store.update(projectID: "nope") { $0.pinned = true }
        }
    }

    @Test func deleteCascadesToScriptsAndProcesses() async throws {
        let store = try Store.inMemory()
        let project = fullProject()
        try await store.upsert(project)

        try await store.deleteProject(id: project.id)

        let db = store.db
        #expect(try db.query("SELECT COUNT(*) AS n FROM project_scripts").first?.int("n") == 0)
        #expect(try db.query("SELECT COUNT(*) AS n FROM project_processes").first?.int("n") == 0)
        #expect(try await store.project(id: project.id) == nil)
    }

    @Test func deleteOfAMissingProjectThrows() async throws {
        let store = try Store.inMemory()

        await #expect(throws: StoreError.projectNotFound("nope")) {
            try await store.deleteProject(id: "nope")
        }
    }

    @Test func lookupByPathAndDomain() async throws {
        let store = try Store.inMemory()
        let project = fullProject()
        try await store.upsert(project)

        #expect(try await store.project(path: project.path)?.id == project.id)
        #expect(try await store.project(domain: project.localDomain)?.id == project.id)
        #expect(try await store.project(path: "/nowhere") == nil)
        #expect(try await store.project(domain: "nowhere.orbit.test") == nil)
    }

    @Test func projectsAreOrderedByNameIgnoringCase() async throws {
        let store = try Store.inMemory()
        for (index, name) in ["beta", "Alpha", "gamma"].enumerated() {
            var project = fullProject(id: "0000000\(index)-0000-0000-0000-000000000000")
            project.name = name
            try await store.upsert(project)
        }

        #expect(try await store.projects().map(\.name) == ["Alpha", "beta", "gamma"])
    }

    @Test func unknownStatusInARowThrowsInsteadOfDefaulting() async throws {
        let store = try Store.inMemory()
        let project = fullProject()
        try await store.upsert(project)
        try store.db.run("UPDATE projects SET status = 'hibernating' WHERE id = ?;", [.text(project.id)])

        await #expect(throws: StoreError.corruptRow(table: "projects", column: "status", value: "hibernating")) {
            try await store.project(id: project.id)
        }
        await #expect(throws: StoreError.corruptRow(table: "projects", column: "status", value: "hibernating")) {
            try await store.update(projectID: project.id) { $0.pinned = false }
        }
    }

    @Test func unknownProcessRoleThrows() async throws {
        let store = try Store.inMemory()
        let project = fullProject()
        try await store.upsert(project)
        try store.db.run("UPDATE project_processes SET role = 'worker' WHERE project_id = ?;", [.text(project.id)])

        await #expect(throws: StoreError.corruptRow(table: "project_processes", column: "role", value: "worker")) {
            try await store.projects()
        }
    }
}
