import Foundation
import Testing

@testable import Core

@Suite struct ParkedScannerTests {
    @Test func aManifestOrAGitFolderMarksAProject() throws {
        let directory = try TemporaryDirectory()
        let empty = try directory.makeDirectory("empty")
        let manifest = try directory.makeDirectory("manifest")
        try directory.write("{}", to: "manifest/package.json")
        let git = try directory.makeDirectory("git/.git")

        #expect(!ParkedScanner.isProjectDirectory(empty))
        #expect(ParkedScanner.isProjectDirectory(manifest))
        #expect(ParkedScanner.isProjectDirectory((git as NSString).deletingLastPathComponent))
    }

    @Test(arguments: [
        "package.json", "composer.json", "go.mod", "pyproject.toml", "requirements.txt", "Pipfile", "manage.py",
    ])
    func everyKnownManifestMarksAProject(manifest: String) throws {
        let directory = try TemporaryDirectory()
        try directory.write("", to: "app/\(manifest)")

        #expect(ParkedScanner.isProjectDirectory("\(directory.path)/app"))
    }

    @Test func discoveryIgnoresNoiseAndNonProjectsAndSorts() throws {
        let directory = try TemporaryDirectory()
        try directory.write("{}", to: "zeta/composer.json")
        try directory.write("{}", to: "alpha/composer.json")
        try directory.makeDirectory("node_modules")
        try directory.write("{}", to: "node_modules/package.json")
        try directory.makeDirectory(".cache")
        try directory.makeDirectory("random-notes")

        let found = ParkedScanner.discoverProjects(in: directory.path)

        #expect(found == ["\(directory.path)/alpha", "\(directory.path)/zeta"])
    }

    @Test func discoveryOfAMissingFolderIsEmpty() {
        #expect(ParkedScanner.discoverProjects(in: "/does/not/exist").isEmpty)
    }

    @Test func storeKeepsParkedFoldersUniqueAndOrdered() async throws {
        let directory = try TemporaryDirectory()
        let path = "\(directory.path)/orbit.sqlite"
        let store = try Store(path: path)
        try await store.addParkedFolder("/a")
        try await store.addParkedFolder("/a")
        try await store.addParkedFolder("/b")
        #expect(try await store.parkedFolders() == ["/a", "/b"])

        let reopened = try Store(path: path)
        #expect(try await reopened.parkedFolders() == ["/a", "/b"])
        try await reopened.removeParkedFolder("/a")
        #expect(try await reopened.parkedFolders() == ["/b"])
    }
}
