import Foundation

public enum ProjectEnvFile {
    public static func read(directory: String) -> [String: String] {
        DotEnv.read(file: path(in: directory))
    }

    public static func write(_ values: [String: String], directory: String) throws {
        let contents = values.keys.sorted().map { "\($0)=\(values[$0] ?? "")\n" }.joined()
        let path = path(in: directory)
        let descriptor = open(path, O_WRONLY | O_CREAT | O_TRUNC, 0o644)
        guard descriptor >= 0 else {
            throw CocoaError.error(.fileWriteUnknown, userInfo: [NSFilePathErrorKey: path])
        }
        let handle = FileHandle(fileDescriptor: descriptor, closeOnDealloc: true)
        try handle.write(contentsOf: Data(contents.utf8))
    }

    private static func path(in directory: String) -> String {
        (directory as NSString).appendingPathComponent(DotEnv.files[0])
    }
}
