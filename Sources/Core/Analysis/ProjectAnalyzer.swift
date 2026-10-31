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
        let canonical = CanonicalPath.of(trimmed)
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: canonical, isDirectory: &isDirectory), isDirectory.boolValue
        else {
            throw AnalysisError.notADirectory(canonical)
        }
        let name = URL(fileURLWithPath: canonical).lastPathComponent

        let composer = ManifestReader.composer(in: canonical)
        if let composer, composer.isLaravel {
            return analyzeLaravel(composer, name: name, path: canonical)
        }
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
        let manifest = try ManifestReader.package(in: canonical)
        if manifest == nil, PythonProjectAnalyzer.isPythonProject(canonical) {
            return PythonProjectAnalyzer(resolver: resolver).analyze(at: canonical)
        }
        if manifest == nil, !FileManager.default.fileExists(atPath: canonical + "/composer.json") {
            if let monorepo = MonorepoAnalyzer(resolver: resolver).analyze(at: canonical) {
                return monorepo
            }
        }
        guard let manifest else {
            throw AnalysisError.noSupportedProject(canonical)
        }
        return analyzeNode(manifest, name: name, path: canonical)
    }

    private func analyzeLaravel(_ composer: ComposerManifest, name: String, path: String) -> ProjectAnalysis {
        let phpVersion = resolver.php(constraint: composer.phpConstraint)
        var analysis = ProjectAnalysis(
            name: name,
            path: path,
            packageManager: "composer",
            framework: "laravel",
            runtimeKind: .phpFPM,
            phpVersion: phpVersion,
            processes: [ProcessSpec(role: .backend, kind: .phpFPM, phpVersion: phpVersion)]
        )
        if let frontend = inertiaFrontend(at: path) {
            analysis.framework = "laravel-inertia"
            analysis.nodeVersion = frontend.nodeVersion
            analysis.processes.append(frontend.spec)
        }
        return analysis
    }

    private func inertiaFrontend(at path: String) -> (spec: ProcessSpec, nodeVersion: String?)? {
        guard let manifest = try? ManifestReader.package(in: path) else {
            return nil
        }
        let dependencies = manifest.allDependencyNames
        guard
            dependencies.contains("vite") || dependencies.contains("laravel-vite-plugin")
                || ViteDetector.hasViteConfig(in: path)
        else {
            return nil
        }
        let packageManager = PackageManagerDetector.node(in: path, declared: manifest.packageManager)
        let command = DevCommand.suggest(scripts: manifest.scripts ?? [:], packageManager: packageManager)
        return (
            ProcessSpec(role: .frontend, kind: .command, command: command),
            resolver.node(in: path, engines: manifest.engines?["node"])
        )
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
