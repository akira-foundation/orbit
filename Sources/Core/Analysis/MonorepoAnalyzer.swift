import Foundation

struct MonorepoAnalyzer: Sendable {
    static let ambiguousReason = "multiple Laravel backend candidates found in subfolders; not auto-wiring"

    private static let skippedDirectories: Set<String> = ["node_modules", "vendor"]

    let resolver: RuntimeVersionResolver

    func analyze(at path: String) -> ProjectAnalysis? {
        let base = URL(fileURLWithPath: path)
        let names =
            ((try? FileManager.default.contentsOfDirectory(atPath: path)) ?? []).sorted()
        var backends: [ProcessSpec] = []
        var frontends: [(spec: ProcessSpec, nodeVersion: String?)] = []

        for name in names where !name.hasPrefix(".") && !Self.skippedDirectories.contains(name) {
            let sub = base.appendingPathComponent(name, isDirectory: true).path
            var isDirectory: ObjCBool = false
            guard FileManager.default.fileExists(atPath: sub, isDirectory: &isDirectory), isDirectory.boolValue else {
                continue
            }
            if let composer = ManifestReader.composer(in: sub), composer.isLaravel {
                backends.append(
                    ProcessSpec(
                        role: .backend,
                        kind: .phpFPM,
                        workDirectory: name,
                        phpVersion: resolver.php(constraint: composer.phpConstraint)
                    ))
                continue
            }
            guard let manifest = try? ManifestReader.package(in: sub),
                manifest.allDependencyNames.contains("vite") || ViteDetector.hasViteConfig(in: sub)
            else {
                continue
            }
            let packageManager = PackageManagerDetector.node(in: sub, declared: manifest.packageManager)
            let command = DevCommand.suggest(scripts: manifest.scripts ?? [:], packageManager: packageManager)
            frontends.append(
                (
                    ProcessSpec(role: .frontend, kind: .command, workDirectory: name, command: command),
                    resolver.node(in: sub, engines: manifest.engines?["node"])
                ))
        }

        let folderName = base.lastPathComponent
        if backends.count > 1 {
            return ProjectAnalysis(name: folderName, path: path, ambiguousReason: Self.ambiguousReason)
        }
        guard let backend = backends.first else {
            return nil
        }
        var analysis = ProjectAnalysis(
            name: folderName,
            path: path,
            framework: "monorepo",
            runtimeKind: backend.kind,
            phpVersion: backend.phpVersion,
            processes: [backend]
        )
        if frontends.count == 1, let frontend = frontends.first {
            analysis.processes.append(frontend.spec)
            analysis.nodeVersion = frontend.nodeVersion
        }
        return analysis
    }
}
