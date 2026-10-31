import Foundation

struct PythonProjectAnalyzer: Sendable {
    let resolver: RuntimeVersionResolver

    private static let markers = [
        "pyproject.toml", "requirements.txt", "Pipfile", "manage.py", "setup.py", "asgi.py", "wsgi.py",
    ]
    private static let knownPackages = ["django", "fastapi", "flask", "starlette", "uvicorn", "gunicorn"]
    private static let asgiModules = ["main", "app", "asgi", "server"]
    private static let entryFiles = ["main.py", "app.py", "run.py"]

    static func isPythonProject(_ dir: String) -> Bool {
        markers.contains { exists($0, in: dir) }
    }

    func analyze(at dir: String) -> ProjectAnalysis {
        let framework = Self.framework(in: dir, dependencies: Self.dependencies(in: dir))
        let (command, port) = Self.devCommand(in: dir, framework: framework)
        return ProjectAnalysis(
            name: URL(fileURLWithPath: dir).lastPathComponent,
            path: dir,
            packageManager: PackageManagerDetector.python(in: dir),
            framework: framework,
            devCommand: command,
            devPort: port,
            runtimeKind: .python,
            pythonVersion: resolver.python(in: dir)
        )
    }

    private static func exists(_ name: String, in dir: String) -> Bool {
        FileManager.default.fileExists(atPath: URL(fileURLWithPath: dir).appendingPathComponent(name).path)
    }

    private static func read(_ name: String, in dir: String) -> String? {
        try? String(contentsOfFile: URL(fileURLWithPath: dir).appendingPathComponent(name).path, encoding: .utf8)
    }

    private static func dependencies(in dir: String) -> Set<String> {
        var found = Set<String>()
        if let requirements = read("requirements.txt", in: dir) {
            for line in requirements.split(separator: "\n", omittingEmptySubsequences: false) {
                if let name = requirementName(String(line)) {
                    found.insert(name)
                }
            }
        }
        if let pyproject = read("pyproject.toml", in: dir)?.lowercased() {
            for package in knownPackages where pyproject.contains(package) {
                found.insert(package)
            }
        }
        return found
    }

    static func requirementName(_ line: String) -> String? {
        let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty, !trimmed.hasPrefix("#"), !trimmed.hasPrefix("-") else {
            return nil
        }
        let end = trimmed.firstIndex { "=<>!~[ ;".contains($0) } ?? trimmed.endIndex
        let name = trimmed[..<end].trimmingCharacters(in: .whitespaces).lowercased()
        return name.isEmpty ? nil : name
    }

    private static func framework(in dir: String, dependencies: Set<String>) -> String {
        if exists("manage.py", in: dir) {
            return "django"
        }
        return ["django", "fastapi", "starlette", "flask"].first(where: dependencies.contains) ?? "python"
    }

    private static func devCommand(in dir: String, framework: String) -> (String, Int) {
        switch framework {
        case "django":
            return ("python manage.py runserver 127.0.0.1:$PORT", 8000)
        case "flask":
            return ("flask run --host 127.0.0.1 --port $PORT", 5000)
        case "fastapi", "starlette":
            return ("uvicorn \(asgiTarget(in: dir)) --host 127.0.0.1 --port $PORT", 8000)
        default:
            let entry = entryFiles.first { exists($0, in: dir) } ?? "main.py"
            return ("python \(entry)", 8000)
        }
    }

    private static func asgiTarget(in dir: String) -> String {
        asgiModules.first { exists($0 + ".py", in: dir) }.map { $0 + ":app" } ?? "main:app"
    }
}
