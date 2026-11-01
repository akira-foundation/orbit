import Synchronization

@testable import Core

final class ScriptedArgumentSource: Sendable {
    private struct State {
        var failures: [Int32]
        var calls = 0
    }

    private static let payload: [UInt8] =
        withUnsafeBytes(of: Int32(3)) { Array($0) } + Array("/bin/sh\0\0sh\0-c\0echo ready\0".utf8)

    private let state: Mutex<State>

    init(failures: [Int32]) {
        state = Mutex(State(failures: failures))
    }

    var calls: Int {
        state.withLock { $0.calls }
    }

    func read(_ pid: Int32, _ buffer: inout [UInt8], _ size: inout Int) -> Int32 {
        let failure = state.withLock { state -> Int32? in
            state.calls += 1
            return state.failures.isEmpty ? nil : state.failures.removeFirst()
        }
        if let failure { return failure }
        buffer.replaceSubrange(0..<Self.payload.count, with: Self.payload)
        size = Self.payload.count
        return 0
    }
}
