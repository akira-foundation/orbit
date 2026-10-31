import Foundation

final class TemporaryDirectory: Sendable {
    let path: String

    init() throws {
        let url = FileManager.default.temporaryDirectory
            .appendingPathComponent("orbit-tests-\(UUID().uuidString)", isDirectory: true)
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        path = url.path
    }

    func write(_ contents: String, to relativePath: String) throws {
        let url = URL(fileURLWithPath: path).appendingPathComponent(relativePath)
        try FileManager.default.createDirectory(
            at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        try contents.write(to: url, atomically: true, encoding: .utf8)
    }

    @discardableResult
    func makeDirectory(_ relativePath: String) throws -> String {
        let url = URL(fileURLWithPath: path).appendingPathComponent(relativePath, isDirectory: true)
        try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
        return url.path
    }

    deinit {
        try? FileManager.default.removeItem(atPath: path)
    }
}
