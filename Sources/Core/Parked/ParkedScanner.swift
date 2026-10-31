import Foundation

public enum ParkedScanner {
    private static let manifestFiles = [
        "package.json", "composer.json", "go.mod", "pyproject.toml", "requirements.txt", "Pipfile", "manage.py",
    ]
    private static let skippedDirectories: Set<String> = ["node_modules", "vendor", "dist", "build"]

    public static func isProjectDirectory(_ path: String) -> Bool {
        let root = URL(fileURLWithPath: path, isDirectory: true)
        if manifestFiles.contains(where: { isFile(root.appendingPathComponent($0).path) }) {
            return true
        }
        return isDirectory(root.appendingPathComponent(".git").path)
    }

    public static func discoverProjects(in folder: String) -> [String] {
        let root = URL(fileURLWithPath: folder, isDirectory: true)
        guard let names = try? FileManager.default.contentsOfDirectory(atPath: folder) else { return [] }
        return names.sorted().compactMap { name -> String? in
            guard !name.hasPrefix("."), !skippedDirectories.contains(name) else { return nil }
            let path = root.appendingPathComponent(name, isDirectory: true).path
            return isDirectory(path) && isProjectDirectory(path) ? path : nil
        }
    }

    private static func isDirectory(_ path: String) -> Bool {
        var isDirectory: ObjCBool = false
        return FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory) && isDirectory.boolValue
    }

    private static func isFile(_ path: String) -> Bool {
        var isDirectory: ObjCBool = false
        return FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory) && !isDirectory.boolValue
    }
}
