import Foundation
import Testing

@testable import Core

struct LaravelAnalysisTests {
    private static func composer(php: String? = "^8.3", extra: String = "") -> String {
        let phpEntry = php.map { #""php":"\#($0)","# } ?? ""
        return #"{"name":"acme/app","require":{\#(phpEntry)"laravel/framework":"^11.0"\#(extra)}}"#
    }

    @Test func laravelProjectResolvesFrameworkRuntimeAndPHP() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(), to: "composer.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "laravel")
        #expect(analysis.runtimeKind == .phpFPM)
        #expect(analysis.phpVersion != nil)
        #expect(analysis.packageManager == "composer")
        #expect(analysis.processes == [ProcessSpec(role: .backend, kind: .phpFPM, phpVersion: analysis.phpVersion)])
    }

    @Test func laravelUsesFolderNameNotComposerName() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(php: "^8.5"), to: "nosferry.com/composer.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path + "/nosferry.com")

        #expect(analysis.name == "nosferry.com")
    }

    @Test func phpConstraintPicksTheNewestMatchingBundledVersion() throws {
        let caret = try TemporaryDirectory()
        try caret.write(Self.composer(php: "^8.3"), to: "composer.json")
        let tilde = try TemporaryDirectory()
        try tilde.write(Self.composer(php: "~8.4.0"), to: "composer.json")

        #expect(try ProjectAnalyzer().analyze(path: caret.path).phpVersion == "8.5.8")
        #expect(try ProjectAnalyzer().analyze(path: tilde.path).phpVersion == "8.4.23")
    }

    @Test func commaSeparatedPHPConstraintExcludesTheUpperBound() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(php: ">=8.2,<8.4"), to: "composer.json")

        #expect(try ProjectAnalyzer().analyze(path: dir.path).phpVersion == "8.3.32")
    }

    @Test func inertiaDetectsFrontendCompanion() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(), to: "composer.json")
        try dir.write(
            #"{"name":"app","scripts":{"dev":"vite"},"devDependencies":{"vite":"^5.0.0","laravel-vite-plugin":"^1.0.0"}}"#,
            to: "package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "laravel-inertia")
        #expect(analysis.nodeVersion != nil)
        #expect(analysis.processes.count == 2)
        #expect(analysis.processes[0].role == .backend)
        #expect(analysis.processes[0].kind == .phpFPM)
        #expect(analysis.processes[1].role == .frontend)
        #expect(analysis.processes[1].kind == .command)
        #expect(analysis.processes[1].command == "npm run dev")
    }

    @Test func viteConfigAloneMakesAnInertiaFrontend() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(), to: "composer.json")
        try dir.write(#"{"name":"app"}"#, to: "package.json")
        try dir.write("export default {}", to: "vite.config.ts")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "laravel-inertia")
        #expect(analysis.processes.count == 2)
    }

    @Test func laravelWithoutViteHasNoFrontendCompanion() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(), to: "composer.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "laravel")
        #expect(analysis.processes.count == 1)
    }

    @Test func invalidComposerFallsBackToNode() throws {
        let dir = try TemporaryDirectory()
        try dir.write("{not json", to: "composer.json")
        try dir.write(#"{"dependencies":{"next":"14"}}"#, to: "package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "nextjs")
        #expect(analysis.runtimeKind == .command)
    }

    @Test func monorepoResolvesBackendAndFrontend() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(), to: "backend/composer.json")
        try dir.write(
            #"{"name":"web","scripts":{"dev":"vite"},"dependencies":{"vite":"^5.0.0"}}"#,
            to: "frontend/package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "monorepo")
        #expect(!analysis.isAmbiguous)
        #expect(analysis.runtimeKind == .phpFPM)
        #expect(analysis.nodeVersion != nil)
        #expect(analysis.processes.count == 2)
        #expect(analysis.processes[0].workDirectory == "backend")
        #expect(analysis.processes[0].role == .backend)
        #expect(analysis.processes[1].workDirectory == "frontend")
        #expect(analysis.processes[1].command == "npm run dev")
    }

    @Test func monorepoWithTwoBackendsIsAmbiguous() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(php: nil), to: "api/composer.json")
        try dir.write(Self.composer(php: nil), to: "admin/composer.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.isAmbiguous)
        #expect(
            analysis.ambiguousReason
                == "multiple Laravel backend candidates found in subfolders; not auto-wiring")
        #expect(analysis.processes.isEmpty)
    }

    @Test func monorepoIgnoresVendorAndHiddenFolders() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(), to: "backend/composer.json")
        try dir.write(Self.composer(), to: "vendor/x/composer.json")
        try dir.write(Self.composer(), to: "vendor/composer.json")
        try dir.write(Self.composer(), to: ".cache/composer.json")
        try dir.write(Self.composer(), to: "node_modules/composer.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(!analysis.isAmbiguous)
        #expect(analysis.framework == "monorepo")
        #expect(analysis.processes.map(\.workDirectory) == ["backend"])
    }

    @Test func monorepoWithTwoFrontendsWiresOnlyTheBackend() throws {
        let dir = try TemporaryDirectory()
        try dir.write(Self.composer(), to: "backend/composer.json")
        try dir.write(#"{"dependencies":{"vite":"5"}}"#, to: "web/package.json")
        try dir.write(#"{"dependencies":{"vite":"5"}}"#, to: "admin/package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.processes.count == 1)
        #expect(analysis.nodeVersion == nil)
    }

    @Test func folderWithoutManifestIsNotSupported() throws {
        let dir = try TemporaryDirectory()
        try dir.write("hello", to: "docs/readme.txt")

        let error = #expect(throws: AnalysisError.self) {
            try ProjectAnalyzer().analyze(path: dir.path)
        }

        guard case .noSupportedProject = error else {
            Issue.record("expected noSupportedProject, got \(String(describing: error))")
            return
        }
    }
}
