import Foundation

public enum ExecutableLookup {
    public static func resolve(_ name: String, searchPath: String) -> String {
        if name.contains("/") {
            return name
        }
        for directory in searchPath.split(separator: ":") {
            let candidate = (String(directory) as NSString).appendingPathComponent(name)
            if isExecutableFile(candidate) {
                return candidate
            }
        }
        return name
    }

    private static func isExecutableFile(_ path: String) -> Bool {
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory), !isDirectory.boolValue else {
            return false
        }
        let permissions = try? FileManager.default.attributesOfItem(atPath: path)[.posixPermissions] as? Int
        return (permissions ?? 0) & 0o111 != 0
    }
}
