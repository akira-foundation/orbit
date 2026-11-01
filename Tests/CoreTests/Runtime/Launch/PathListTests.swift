import Foundation
import Testing

@testable import Core

struct PathListTests {
    @Test func mergeDedupsAndKeepsFirstOrder() {
        #expect(PathList.merge("/usr/bin:/bin", "/opt/homebrew/bin:/usr/bin") == "/usr/bin:/bin:/opt/homebrew/bin")
    }

    @Test func mergeDropsEmptyEntries() {
        #expect(PathList.merge("/a::/b", ":/c:") == "/a:/b:/c")
    }

    @Test func valueUsesTheEnvironmentPath() {
        #expect(PathList.value(in: EnvironmentList(["PATH=/x:/y"])) == "/x:/y")
    }

    @Test func valueFallsBackToTheProcessPath() {
        let expected = ProcessInfo.processInfo.environment["PATH"] ?? ""
        #expect(PathList.value(in: EnvironmentList(["HOME=/h"])) == expected)
    }

    @Test func lookupFindsExecutableInLaterDirectory() throws {
        let directory = try TemporaryDirectory()
        try directory.write("#!/bin/sh\n", to: "bun")
        try directory.write("x", to: "data")
        try directory.makeDirectory("tool")
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: directory.path + "/bun")
        try FileManager.default.setAttributes([.posixPermissions: 0o644], ofItemAtPath: directory.path + "/data")
        let searchPath = "/nope:" + directory.path
        #expect(ExecutableLookup.resolve("bun", searchPath: searchPath) == directory.path + "/bun")
        #expect(ExecutableLookup.resolve("/abs/bun", searchPath: searchPath) == "/abs/bun")
        #expect(ExecutableLookup.resolve("missing", searchPath: searchPath) == "missing")
        #expect(ExecutableLookup.resolve("data", searchPath: searchPath) == "data")
        #expect(ExecutableLookup.resolve("bin/x", searchPath: searchPath) == "bin/x")
        #expect(ExecutableLookup.resolve("tool", searchPath: searchPath) == "tool")
    }
}
