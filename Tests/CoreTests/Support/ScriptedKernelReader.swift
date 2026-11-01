import Synchronization

@testable import Core

final class ScriptedKernelReader: KernelProcessReading {
    private struct State {
        var snapshots: [KernelProcessSnapshot]
        var arguments: [String]
        var argumentReads = 0
    }

    private let state: Mutex<State>
    private let table: [KernelProcessSnapshot]
    private let vanished: Set<Int32>
    private let tableFailure: String?

    init(
        snapshots: [KernelProcessSnapshot] = [], table: [KernelProcessSnapshot] = [], vanished: Set<Int32> = [],
        tableFailure: String? = nil, arguments: [String] = []
    ) {
        state = Mutex(State(snapshots: snapshots, arguments: arguments))
        self.table = table
        self.vanished = vanished
        self.tableFailure = tableFailure
    }

    var argumentReads: Int {
        state.withLock { $0.argumentReads }
    }

    func snapshot(_ pid: Int32) throws -> KernelProcessSnapshot {
        if let scripted = state.withLock({ $0.snapshots.isEmpty ? nil : $0.snapshots.removeFirst() }) {
            return scripted
        }
        guard !vanished.contains(pid), let listed = table.first(where: { $0.pid == pid }) else {
            throw RuntimeError.processGone
        }
        return listed
    }

    func group(_ pgid: Int32) throws -> [KernelProcessSnapshot] {
        table.filter { $0.pgid == pgid }
    }

    func all() throws -> [KernelProcessSnapshot] {
        if let tableFailure { throw RuntimeError.message(tableFailure) }
        return table
    }

    func arguments(_ pid: Int32) throws -> String {
        state.withLock { state in
            state.argumentReads += 1
            return state.arguments.isEmpty ? "fixture \(pid)" : state.arguments.removeFirst()
        }
    }
}
