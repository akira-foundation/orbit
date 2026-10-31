import Foundation

enum PackageManagerDetector {
    private static let lockfiles = [
        ("pnpm-lock.yaml", "pnpm"),
        ("bun.lockb", "bun"),
        ("bun.lock", "bun"),
        ("yarn.lock", "yarn"),
        ("package-lock.json", "npm"),
    ]

    static func node(in directory: String, declared: String?) -> String {
        if let declared, !declared.isEmpty {
            if let separator = declared.lastIndex(of: "@"), separator != declared.startIndex {
                return String(declared[..<separator])
            }
            return declared.trimmingCharacters(in: .whitespaces)
        }
        let base = URL(fileURLWithPath: directory)
        let match = lockfiles.first { file, _ in
            FileManager.default.fileExists(atPath: base.appendingPathComponent(file).path)
        }
        return match?.1 ?? "npm"
    }

    static func python(in directory: String) -> String {
        let base = URL(fileURLWithPath: directory)
        let files = [("uv.lock", "uv"), ("poetry.lock", "poetry"), ("Pipfile.lock", "pipenv"), ("Pipfile", "pipenv")]
        if let match = files.first(where: {
            FileManager.default.fileExists(atPath: base.appendingPathComponent($0.0).path)
        }) {
            return match.1
        }
        let pyproject = try? String(contentsOfFile: base.appendingPathComponent("pyproject.toml").path, encoding: .utf8)
        if let pyproject {
            if pyproject.contains("[tool.uv") {
                return "uv"
            }
            if pyproject.contains("[tool.poetry") {
                return "poetry"
            }
        }
        return "pip"
    }
}
