import Foundation

public enum ProjectEnvFile {
    public static func read(directory: String) -> [String: String] {
        guard let data = FileManager.default.contents(atPath: path(in: directory)) else {
            return [:]
        }
        var values: [String: String] = [:]
        for pair in DotEnv.parse(String(decoding: data, as: UTF8.self)) {
            values[pair.key] = pair.value
        }
        return values
    }

    public static func write(_ values: [String: String], directory: String) throws {
        let contents = values.keys.sorted().map { "\($0)=\(values[$0] ?? "")\n" }.joined()
        let url = URL(fileURLWithPath: path(in: directory))
        try contents.write(to: url, atomically: true, encoding: .utf8)
        try FileManager.default.setAttributes([.posixPermissions: 0o644], ofItemAtPath: url.path)
    }

    private static func path(in directory: String) -> String {
        (directory as NSString).appendingPathComponent(".env")
    }
}
