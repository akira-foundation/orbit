import Foundation
import Testing

@testable import Core

struct DotEnvTests {
    @Test func parseHandlesExportCommentsAndQuoteRuns() {
        let pairs = DotEnv.parse("# c\n\nexport A=1\nB = \"two\"\nC='\"x\"'\n=skip\nD=a=b\n")
        #expect(pairs.map(\.key) == ["A", "B", "C", "D"])
        #expect(pairs.map(\.value) == ["1", "two", "x", "a=b"])
    }

    @Test func parseKeepsALineOf65535Bytes() {
        let line = "BIG=" + String(repeating: "x", count: 65_535 - 4)
        let pairs = DotEnv.parse("A=1\n\(line)\nB=2\n")
        #expect(pairs.map(\.key) == ["A", "BIG", "B"])
    }

    @Test func parseStopsAtALineOf65536Bytes() {
        let line = "BIG=" + String(repeating: "x", count: 65_536 - 4)
        let pairs = DotEnv.parse("A=1\n\(line)\nB=2\n")
        #expect(pairs.map(\.key) == ["A"])
    }

    @Test func parseReadsCRLFLines() {
        let pairs = DotEnv.parse("A=1\r\nB=2\r\n")
        #expect(pairs.map(\.key) == ["A", "B"])
        #expect(pairs.map(\.value) == ["1", "2"])
    }

    @Test func loadMergesFilesInOrderLaterWins() throws {
        let directory = try TemporaryDirectory()
        try directory.write("A=1\nB=1\n", to: ".env")
        try directory.write("B=2\n", to: ".env.local")
        try directory.write("B=4\n", to: ".env.development.local")
        #expect(DotEnv.load(directory: directory.path) == ["A": "1", "B": "4"])
    }

    @Test func loadOrdersDevelopmentBeforeDevelopmentLocal() throws {
        let directory = try TemporaryDirectory()
        try directory.write("K=local\n", to: ".env.development.local")
        try directory.write("K=development\n", to: ".env.development")
        try directory.write("K=env\n", to: ".env.local")
        #expect(DotEnv.load(directory: directory.path) == ["K": "local"])
        try FileManager.default.removeItem(atPath: directory.path + "/.env.development.local")
        #expect(DotEnv.load(directory: directory.path) == ["K": "development"])
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

    @Test func writeKeepsTheModeOfAnExistingFile() throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/.env"
        try directory.write("OLD=1\n", to: ".env")
        try FileManager.default.setAttributes([.posixPermissions: 0o600], ofItemAtPath: path)
        try ProjectEnvFile.write(["NEW": "2"], directory: directory.path)
        let permissions = try FileManager.default.attributesOfItem(atPath: path)[.posixPermissions] as? Int
        #expect(permissions == 0o600)
        #expect(try String(contentsOfFile: path, encoding: .utf8) == "NEW=2\n")
    }

    @Test func writeFollowsASymlinkedFile() throws {
        let directory = try TemporaryDirectory()
        try directory.write("OLD=1\n", to: "real.env")
        try FileManager.default.createSymbolicLink(
            atPath: directory.path + "/.env", withDestinationPath: directory.path + "/real.env")
        try ProjectEnvFile.write(["NEW": "2"], directory: directory.path)
        let attributes = try FileManager.default.attributesOfItem(atPath: directory.path + "/.env")
        #expect(attributes[.type] as? FileAttributeType == .typeSymbolicLink)
        #expect(try String(contentsOfFile: directory.path + "/real.env", encoding: .utf8) == "NEW=2\n")
    }

    @Test func readProjectEnvMissingFileIsEmpty() throws {
        let directory = try TemporaryDirectory()
        #expect(ProjectEnvFile.read(directory: directory.path).isEmpty)
    }
}
