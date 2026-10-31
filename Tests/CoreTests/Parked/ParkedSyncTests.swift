import Foundation
import Testing

@testable import Core

@Suite struct ParkedSyncTests {
    private func makeSync(
        _ directory: TemporaryDirectory, analyzer: any ProjectAnalyzing = ProjectAnalyzer()
    ) async throws -> (ParkedSync, Store, ProjectRegistry) {
        let store = try Store.inMemory()
        let registry = ProjectRegistry(store: store, analyzer: analyzer, environment: .production)
        try await store.addParkedFolder("\(directory.path)/sites")
        return (ParkedSync(store: store, registry: registry), store, registry)
    }

    private func nodeProject(_ directory: TemporaryDirectory, _ name: String, in folder: String = "sites") throws {
        try directory.write(
            #"{"name":"\#(name)","dependencies":{"@nestjs/core":"^10"}}"#, to: "\(folder)/\(name)/package.json")
    }

    @Test func syncRegistersDiscoveredProjectsOnce() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "shop")
        let (sync, store, _) = try await makeSync(directory)

        let first = try await sync.run()
        let second = try await sync.run()

        #expect(first.added.count == 1)
        #expect(second.added.isEmpty)
        #expect(try await store.projects().first?.parkedFolder == "\(directory.path)/sites")
    }

    @Test func syncRemovesProjectsWhoseFolderIsGone() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "shop")
        let (sync, store, _) = try await makeSync(directory)
        let added = try await sync.run().added

        try FileManager.default.removeItem(atPath: "\(directory.path)/sites/shop")
        let result = try await sync.run()

        #expect(result.removed.map(\.id) == added.map(\.id))
        #expect(try await store.projects().isEmpty)
    }

    @Test func syncKeepsProjectsWhenTheWholeParkedFolderIsGone() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "shop")
        try nodeProject(directory, "blog")
        let (sync, store, _) = try await makeSync(directory)
        let added = try await sync.run().added

        try FileManager.default.removeItem(atPath: "\(directory.path)/sites")
        let result = try await sync.run()

        #expect(result.removed.isEmpty)
        #expect(try await store.projects().map(\.id).sorted() == added.map(\.id).sorted())
    }

    @Test func syncKeepsProjectsWhenTheParkedFolderCannotBeRead() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "shop")
        let (sync, store, _) = try await makeSync(directory)
        _ = try await sync.run()
        let sites = "\(directory.path)/sites"
        defer { try? FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: sites) }

        try FileManager.default.setAttributes([.posixPermissions: 0o000], ofItemAtPath: sites)
        let result = try await sync.run()

        #expect(result.removed.isEmpty)
        #expect(try await store.projects().count == 1)
        withExtendedLifetime(directory) {}
    }

    @Test func syncSkipsAFolderThatFailsAnalysisAndKeepsGoing() async throws {
        let directory = try TemporaryDirectory()
        try directory.write("{", to: "sites/broken/package.json")
        try nodeProject(directory, "ok")
        let (sync, _, _) = try await makeSync(directory)

        let result = try await sync.run()

        #expect(result.added.map(\.name) == ["ok"])
        #expect(result.skipped.map(\.path) == ["\(directory.path)/sites/broken"])
    }

    @Test func syncSkipsAnAmbiguousMonorepo() async throws {
        let directory = try TemporaryDirectory()
        let laravel = #"{"require":{"laravel/framework":"^11.0"}}"#
        try directory.write(laravel, to: "sites/stack/api/composer.json")
        try directory.write(laravel, to: "sites/stack/admin/composer.json")
        try directory.makeDirectory("sites/stack/.git")
        try nodeProject(directory, "ok")
        let (sync, store, _) = try await makeSync(directory)

        let result = try await sync.run()

        #expect(result.added.map(\.name) == ["ok"])
        #expect(result.skipped.map(\.path) == ["\(directory.path)/sites/stack"])
        #expect(try await store.projects().map(\.name) == ["ok"])
    }

    @Test func syncDoesNotTouchProjectsAddedByHand() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "manual", in: "elsewhere")
        let (sync, store, registry) = try await makeSync(directory)
        _ = try await registry.add(path: "\(directory.path)/elsewhere/manual", overwrite: false)

        try FileManager.default.removeItem(atPath: "\(directory.path)/elsewhere/manual")
        let result = try await sync.run()

        #expect(result.removed.isEmpty)
        #expect(try await store.projects().count == 1)
    }

    @Test func removingAParkedFolderKeepsItsProjectsUnmarked() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "shop")
        let (sync, store, _) = try await makeSync(directory)
        _ = try await sync.run()

        try await store.removeParkedFolder("\(directory.path)/sites")

        let projects = try await store.projects()
        #expect(projects.count == 1)
        #expect(projects.first?.parkedFolder == nil)
        #expect(try await store.parkedFolders().isEmpty)
    }

    @Test func slugConflictIsASkipNotAnOverwrite() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "shop", in: "other")
        try nodeProject(directory, "shop")
        let (sync, store, registry) = try await makeSync(directory)
        let manual = try await registry.add(path: "\(directory.path)/other/shop", overwrite: false)

        let result = try await sync.run()

        #expect(result.added.isEmpty)
        #expect(result.skipped.map(\.path) == ["\(directory.path)/sites/shop"])
        #expect(try await store.projects().map(\.id) == [manual.id])
    }

    @Test func aTrailingSlashDoesNotAddTheParkedFolderTwice() async throws {
        let directory = try TemporaryDirectory()
        let (_, store, _) = try await makeSync(directory)

        try await store.addParkedFolder("\(directory.path)/sites/")
        let folders = try await store.parkedFolders()
        try await store.removeParkedFolder("\(directory.path)/sites/")

        #expect(folders == ["\(directory.path)/sites"])
        #expect(try await store.parkedFolders().isEmpty)
    }

    @Test func aParkedFolderAddedThroughASymlinkIsNotReanalysedOnTheNextRun() async throws {
        let directory = try TemporaryDirectory()
        try nodeProject(directory, "shop")
        let link = "\(directory.path)/link"
        try FileManager.default.createSymbolicLink(atPath: link, withDestinationPath: "\(directory.path)/sites")
        let store = try Store.inMemory()
        let registry = ProjectRegistry(store: store, analyzer: ProjectAnalyzer(), environment: .production)
        let sync = ParkedSync(store: store, registry: registry)
        try await store.addParkedFolder(link)
        let first = try await sync.run()

        try directory.write("{", to: "sites/shop/package.json")
        let second = try await sync.run()

        #expect(first.added.map(\.parkedFolder) == ["\(directory.path)/sites"])
        #expect(second == ParkedSyncResult())
    }
}
