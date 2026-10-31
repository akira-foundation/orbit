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
}
