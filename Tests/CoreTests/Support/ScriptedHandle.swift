import Foundation
import Synchronization

@testable import Core

final class ScriptedHandle: SupervisorHandle {
    private struct State {
        var stopCalls = 0
        var exited = false
        var waiters: [CheckedContinuation<Void, Never>] = []
    }

    let identity: ProcessIdentity
    let stdout: FileHandle?
    let stderr: FileHandle?
    private let exitsOnStop: Bool
    private let stdoutWriter: FileHandle
    private let stderrWriter: FileHandle
    private let state = Mutex(State())

    init(pid: Int32, exitsOnStop: Bool) {
        identity = ProcessIdentity(
            pid: pid, pgid: pid, startID: "start-\(pid)", executable: "/fixture/process", arguments: "fixture \(pid)")
        self.exitsOnStop = exitsOnStop
        let output = Pipe()
        let errors = Pipe()
        stdout = output.fileHandleForReading
        stderr = errors.fileHandleForReading
        stdoutWriter = output.fileHandleForWriting
        stderrWriter = errors.fileHandleForWriting
    }

    var stopCalls: Int { state.withLock { $0.stopCalls } }

    func write(_ line: String, to stream: LogStream = .stdout) {
        try? (stream == .stderr ? stderrWriter : stdoutWriter).write(contentsOf: Data((line + "\n").utf8))
    }

    func exit() {
        let waiters = state.withLock { state -> [CheckedContinuation<Void, Never>] in
            guard !state.exited else { return [] }
            state.exited = true
            defer { state.waiters.removeAll() }
            return state.waiters
        }
        try? stdoutWriter.close()
        try? stderrWriter.close()
        waiters.forEach { $0.resume() }
    }

    func stop() throws {
        state.withLock { $0.stopCalls += 1 }
        if exitsOnStop {
            exit()
        }
    }

    func wait() async throws {
        await withCheckedContinuation { continuation in
            let exited = state.withLock { state in
                guard !state.exited else { return true }
                state.waiters.append(continuation)
                return false
            }
            if exited {
                continuation.resume()
            }
        }
    }

    func verifyEmpty(within limit: Duration) async throws {}
}
