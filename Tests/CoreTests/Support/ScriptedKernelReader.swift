import Synchronization

@testable import Core

final class ScriptedKernelReader: KernelProcessReading {
    private struct State {
        var snapshots: [KernelProcessSnapshot]
        var arguments: [String]
        var argumentReads = 0
        var generation = 0
    }

    private let state: Mutex<State>
    private let table: [KernelProcessSnapshot]
    private let vanished: Set<Int32>
    private let tableFailure: String?
    private let argumentFailure: String?
    private let unsettled: Set<Int32>

    init(
        snapshots: [KernelProcessSnapshot] = [], table: [KernelProcessSnapshot] = [], vanished: Set<Int32> = [],
        tableFailure: String? = nil, arguments: [String] = [], argumentFailure: String? = nil,
        unsettled: Set<Int32> = []
    ) {
        self.unsettled = unsettled
        self.argumentFailure = argumentFailure
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
        guard unsettled.contains(pid) else { return listed }
        let generation = state.withLock { state in
            state.generation += 1
            return state.generation
        }
        return KernelProcessSnapshot(
            pid: pid, pgid: listed.pgid, ppid: listed.ppid, startID: "\(generation):999999", command: listed.command
        )
    }

    func group(_ pgid: Int32) throws -> [KernelProcessSnapshot] {
        table.filter { $0.pgid == pgid }
    }

    func all() throws -> [KernelProcessSnapshot] {
        if let tableFailure { throw RuntimeError.message(tableFailure) }
        return table
    }

    func arguments(_ pid: Int32) throws -> String {
        if let argumentFailure { throw RuntimeError.message(argumentFailure) }
        return state.withLock { state in
            state.argumentReads += 1
            return state.arguments.isEmpty ? "fixture \(pid)" : state.arguments.removeFirst()
        }
    }
}
