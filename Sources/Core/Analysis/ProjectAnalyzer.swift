import Foundation

public struct ProjectAnalyzer: ProjectAnalyzing {
    private let resolver: RuntimeVersionResolver

    public init(catalog: RuntimeCatalog = .bundled) {
        resolver = RuntimeVersionResolver(catalog: catalog)
    }

    public func analyze(path: String) throws -> ProjectAnalysis {
        let trimmed = path.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else {
            throw AnalysisError.emptyPath
        }
        let canonical = URL(fileURLWithPath: trimmed).standardizedFileURL.resolvingSymlinksInPath().path
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: canonical, isDirectory: &isDirectory), isDirectory.boolValue
        else {
            throw AnalysisError.notADirectory(canonical)
        }
        let name = URL(fileURLWithPath: canonical).lastPathComponent

        if FileManager.default.fileExists(atPath: canonical + "/go.mod") {
            return ProjectAnalysis(
                name: name,
                path: canonical,
                packageManager: "go modules",
                framework: "go",
                devCommand: "go run main.go",
                devPort: 8080
            )
        }
        guard let manifest = try ManifestReader.package(in: canonical) else {
            throw AnalysisError.noSupportedProject(canonical)
        }
        return analyzeNode(manifest, name: name, path: canonical)
    }

    private func analyzeNode(_ manifest: PackageManifest, name: String, path: String) -> ProjectAnalysis {
        let scripts = manifest.scripts ?? [:]
        let packageManager = PackageManagerDetector.node(in: path, declared: manifest.packageManager)
        let framework = NodeFramework.detect(in: manifest.allDependencyNames)
        return ProjectAnalysis(
            name: name,
            path: path,
            packageManager: packageManager,
            framework: framework?.id,
            devCommand: DevCommand.suggest(scripts: scripts, packageManager: packageManager),
            devPort: DevCommand.port(in: scripts) ?? framework?.port,
            nodeVersion: resolver.node(in: path, engines: manifest.engines?["node"]),
            scripts: scripts
        )
    }
}
