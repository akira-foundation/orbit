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
        var status = stat()
        guard stat(path, &status) == 0, status.st_mode & S_IFMT != S_IFDIR else {
            return false
        }
        return status.st_mode & 0o111 != 0
    }
}
