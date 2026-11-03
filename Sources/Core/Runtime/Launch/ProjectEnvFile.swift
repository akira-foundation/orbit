import Foundation

public enum ProjectEnvFile {
    public static func read(directory: String) -> [String: String] {
        DotEnv.read(file: path(in: directory))
    }

    private static func path(in directory: String) -> String {
        (directory as NSString).appendingPathComponent(DotEnv.files[0])
    }
}
