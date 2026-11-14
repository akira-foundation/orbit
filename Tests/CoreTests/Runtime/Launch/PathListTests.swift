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

    @Test func lookupFirstDirectoryWins() throws {
        let first = try TemporaryDirectory()
        let second = try TemporaryDirectory()
        try makeExecutable("tool", in: first, mode: 0o755)
        try makeExecutable("tool", in: second, mode: 0o755)
        let searchPath = first.path + ":" + second.path
        #expect(ExecutableLookup.resolve("tool", searchPath: searchPath) == first.path + "/tool")
    }

    @Test func lookupSkipsEmptyPathEntries() throws {
        let directory = try TemporaryDirectory()
        try makeExecutable("tool", in: directory, mode: 0o755)
        let searchPath = "::" + directory.path
        #expect(ExecutableLookup.resolve("tool", searchPath: searchPath) == directory.path + "/tool")
    }

    @Test func lookupCountsOwnerOnlyExecute() throws {
        let directory = try TemporaryDirectory()
        try makeExecutable("tool", in: directory, mode: 0o700)
        #expect(ExecutableLookup.resolve("tool", searchPath: directory.path) == directory.path + "/tool")
    }

    @Test func lookupFollowsSymlinksToJudgeTheMode() throws {
        let directory = try TemporaryDirectory()
        try makeExecutable("plain", in: directory, mode: 0o644)
        try makeExecutable("runnable", in: directory, mode: 0o755)
        try FileManager.default.createSymbolicLink(
            atPath: directory.path + "/linkplain", withDestinationPath: directory.path + "/plain")
        try FileManager.default.createSymbolicLink(
            atPath: directory.path + "/linkrunnable", withDestinationPath: directory.path + "/runnable")
        #expect(ExecutableLookup.resolve("linkplain", searchPath: directory.path) == "linkplain")
        #expect(
            ExecutableLookup.resolve("linkrunnable", searchPath: directory.path) == directory.path + "/linkrunnable")
    }

    private func makeExecutable(_ name: String, in directory: TemporaryDirectory, mode: Int) throws {
        try directory.write("x", to: name)
        try FileManager.default.setAttributes([.posixPermissions: mode], ofItemAtPath: directory.path + "/" + name)
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
