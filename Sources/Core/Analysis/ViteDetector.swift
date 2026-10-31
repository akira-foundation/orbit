import Foundation

enum ViteDetector {
    private static let configFiles = ["vite.config.js", "vite.config.ts", "vite.config.mjs", "vite.config.cjs"]

    static func hasViteConfig(in directory: String) -> Bool {
        let base = URL(fileURLWithPath: directory)
        return configFiles.contains {
            FileManager.default.fileExists(atPath: base.appendingPathComponent($0).path)
        }
    }
}
