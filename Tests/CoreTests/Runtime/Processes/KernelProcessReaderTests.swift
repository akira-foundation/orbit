import Darwin
import Testing

@testable import Core

struct KernelProcessReaderTests {
    @Test(arguments: [(1, "100:000001"), (999_999, "100:999999"), (120_000, "100:120000")])
    func startIDPadsMicrosecondsToSixDigits(micros: Int32, expected: String) {
        #expect(KernelProcessReader.startID(timeval(tv_sec: 100, tv_usec: micros)) == expected)
    }

    @Test func argumentsRetryTransientInputOutputErrors() throws {
        let source = ScriptedArgumentSource(failures: [EIO, EIO])
        let reader = KernelProcessReader(argumentSource: source.read)
        #expect(try reader.arguments(getpid()) == "sh -c echo ready")
        #expect(source.calls == 3)
    }

    @Test func argumentsGiveUpAfterRepeatedInputOutputErrors() {
        let source = ScriptedArgumentSource(failures: Array(repeating: EIO, count: 9))
        let reader = KernelProcessReader(argumentSource: source.read)
        #expect(throws: RuntimeError.message("inspect arguments for PID \(getpid()): Input/output error")) {
            try reader.arguments(getpid())
        }
        #expect(source.calls == 8)
    }

    @Test func argumentsOfOwnProcessDoNotHideInvalidArgument() {
        let source = ScriptedArgumentSource(failures: [EINVAL])
        let reader = KernelProcessReader(argumentSource: source.read)
        #expect(throws: RuntimeError.message("inspect arguments for PID \(getpid()): Invalid argument")) {
            try reader.arguments(getpid())
        }
    }

    @Test(arguments: [EINVAL, EPERM])
    func argumentsOfAProcessThatBecameForeignAreEmpty(code: Int32) throws {
        let source = ScriptedArgumentSource(failures: [code])
        let owners = ScriptedOwnerSource(owners: [geteuid(), 0])
        let reader = KernelProcessReader(argumentSource: source.read, ownerSource: owners.read)
        #expect(try reader.arguments(getpid()) == "")
        #expect(source.calls == 1)
    }

    @Test func argumentsOfAProcessStillOursKeepTheError() {
        let source = ScriptedArgumentSource(failures: [EPERM])
        let owners = ScriptedOwnerSource(owners: [geteuid()])
        let reader = KernelProcessReader(argumentSource: source.read, ownerSource: owners.read)
        #expect(throws: RuntimeError.message("inspect arguments for PID \(getpid()): Operation not permitted")) {
            try reader.arguments(getpid())
        }
    }

    @Test func argumentsOfAnotherOwnerAreEmptyWithoutReading() throws {
        let child = try SpawnedTestProcess.spawn("/usr/bin/top", ["-l", "0", "-s", "1"])
        defer { child.terminate() }
        let source = ScriptedArgumentSource(failures: [])
        #expect(try KernelProcessReader(argumentSource: source.read).arguments(child.pid) == "")
        #expect(source.calls == 0)
    }
}
