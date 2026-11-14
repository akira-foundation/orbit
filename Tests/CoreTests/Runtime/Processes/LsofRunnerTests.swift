import Foundation
import Testing

@testable import Core

struct LsofRunnerTests {
    private let directory: TemporaryDirectory

    init() throws {
        directory = try TemporaryDirectory()
    }

    @Test func standardOutputIsReturned() throws {
        #expect(try runner("printf 'p4100\\n'").run(host: "127.0.0.1", port: 80) == "p4100\n")
    }

    @Test func exitOneWithEmptyOutputIsNoOwners() throws {
        #expect(try runner("exit 1").run(host: "127.0.0.1", port: 80) == "")
    }

    @Test func anyStandardErrorIsAnError() throws {
        let runner = try runner("echo 'lsof: WARNING: something' >&2; printf 'p4100\\n'")
        #expect(throws: RuntimeError.message("inspect listener 127.0.0.1:80: lsof: WARNING: something")) {
            try runner.run(host: "127.0.0.1", port: 80)
        }
    }

    @Test func exitOtherThanZeroOrOneIsAnError() throws {
        let runner = try runner("exit 2")
        #expect(throws: RuntimeError.message("inspect listener 127.0.0.1:80: lsof exited with status 2")) {
            try runner.run(host: "127.0.0.1", port: 80)
        }
    }

    @Test func exitOneWithOutputIsAnError() throws {
        let runner = try runner("printf 'p4100\\n'; exit 1")
        #expect(throws: RuntimeError.message("inspect listener 127.0.0.1:80: lsof exited with status 1")) {
            try runner.run(host: "127.0.0.1", port: 80)
        }
    }

    @Test func terminationBySignalIsAnError() throws {
        let runner = try runner("kill -TERM $$")
        #expect(throws: RuntimeError.message("inspect listener 127.0.0.1:80: lsof terminated by signal 15")) {
            try runner.run(host: "127.0.0.1", port: 80)
        }
    }

    @Test func environmentAndArgumentsAreExact() throws {
        let output = try runner(
            "/usr/bin/env | /usr/bin/grep -v -e '^PWD=' -e '^SHLVL=' -e '^_='; printf '%s\\n' \"$@\""
        ).run(host: "127.0.0.2", port: 5173)
        let lines = output.split(separator: "\n").map(String.init)
        #expect(Set(lines.prefix(3)) == ["LC_ALL=C", "LANG=C", "PATH=/usr/bin:/bin:/usr/sbin:/sbin"])
        #expect(Array(lines.dropFirst(3)) == ["-nP", "-a", "-iTCP@127.0.0.2:5173", "-sTCP:LISTEN", "-Fp"])
    }

    @Test(.disabled(if: ThreadSanitizer.isLoaded, ThreadSanitizer.exitTestsUnsupported))
    func callersThatOccupyEveryCooperativeThreadStillFinish() async {
        await #expect(processExitsWith: .success) {
            let watchdog = Thread {
                Thread.sleep(forTimeInterval: 30)
                exit(EXIT_FAILURE)
            }
            watchdog.start()
            guard let directory = try? TemporaryDirectory() else { exit(EXIT_FAILURE) }
            let path = directory.path + "/lsof"
            try? directory.write("#!/bin/sh\n/bin/sleep 0.2\nprintf 'p4100\\n'\n", to: "lsof")
            try? FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: path)
            let runner = LsofRunner(executable: URL(fileURLWithPath: path))
            let callers = ProcessInfo.processInfo.activeProcessorCount * 2
            let outputs = await withTaskGroup(of: String?.self) { group in
                for _ in 0..<callers {
                    group.addTask { try? runner.run(host: "127.0.0.1", port: 80) }
                }
                return await group.reduce(into: [String?]()) { $0.append($1) }
            }
            exit(outputs.allSatisfy { $0 == "p4100\n" } ? EXIT_SUCCESS : EXIT_FAILURE)
        }
    }

    private func runner(_ body: String) throws -> LsofRunner {
        try directory.write("#!/bin/sh\n\(body)\n", to: "lsof")
        let path = directory.path + "/lsof"
        try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: path)
        return LsofRunner(executable: URL(fileURLWithPath: path))
    }
}
