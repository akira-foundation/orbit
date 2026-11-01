import Foundation
import Testing

@testable import Core

struct DotEnvTests {
    @Test func parseHandlesExportCommentsAndQuoteRuns() {
        let pairs = DotEnv.parse("# c\n\nexport A=1\nB = \"two\"\nC='\"x\"'\n=skip\nD=a=b\n")
        #expect(pairs.map(\.key) == ["A", "B", "C", "D"])
        #expect(pairs.map(\.value) == ["1", "two", "x", "a=b"])
    }

    @Test func parseStopsAtALineAbove64KiB() {
        let long = "BIG=" + String(repeating: "x", count: 70_000)
        let pairs = DotEnv.parse("A=1\n\(long)\nB=2\n")
        #expect(pairs.map(\.key) == ["A"])
    }

    @Test func loadMergesFilesInOrderLaterWins() throws {
        let directory = try TemporaryDirectory()
        try directory.write("A=1\nB=1\n", to: ".env")
        try directory.write("B=2\n", to: ".env.local")
        try directory.write("B=4\n", to: ".env.development.local")
        #expect(DotEnv.load(directory: directory.path) == ["A": "1", "B": "4"])
    }

    @Test func loadIgnoresMissingFiles() throws {
        let directory = try TemporaryDirectory()
        #expect(DotEnv.load(directory: directory.path).isEmpty)
    }

    @Test func readWriteProjectEnvRoundTrips() throws {
        let directory = try TemporaryDirectory()
        let values = ["APP_NAME": "orbit", "DB_PORT": "5433"]
        try ProjectEnvFile.write(values, directory: directory.path)
        let path = directory.path + "/.env"
        #expect(ProjectEnvFile.read(directory: directory.path) == values)
        #expect(try String(contentsOfFile: path, encoding: .utf8) == "APP_NAME=orbit\nDB_PORT=5433\n")
        let permissions = try FileManager.default.attributesOfItem(atPath: path)[.posixPermissions] as? Int
        #expect(permissions == 0o644)
    }

    @Test func readProjectEnvMissingFileIsEmpty() throws {
        let directory = try TemporaryDirectory()
        #expect(ProjectEnvFile.read(directory: directory.path).isEmpty)
    }
}
