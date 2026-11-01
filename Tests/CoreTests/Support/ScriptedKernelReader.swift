import Synchronization

@testable import Core

final class ScriptedKernelReader: KernelProcessReading {
    private struct State {
        var snapshots: [KernelProcessSnapshot]
        var argumentReads = 0
    }

    private let state: Mutex<State>
    private let table: [KernelProcessSnapshot]

    init(snapshots: [KernelProcessSnapshot] = [], table: [KernelProcessSnapshot] = []) {
        state = Mutex(State(snapshots: snapshots))
        self.table = table
    }

    var argumentReads: Int {
        state.withLock { $0.argumentReads }
    }

    func snapshot(_ pid: Int32) throws -> KernelProcessSnapshot {
        if let scripted = state.withLock({ $0.snapshots.isEmpty ? nil : $0.snapshots.removeFirst() }) {
            return scripted
        }
        guard let listed = table.first(where: { $0.pid == pid }) else { throw RuntimeError.processGone }
        return listed
    }

    func group(_ pgid: Int32) throws -> [KernelProcessSnapshot] {
        table.filter { $0.pgid == pgid }
    }

    func all() throws -> [KernelProcessSnapshot] {
        table
    }

    func arguments(_ pid: Int32) throws -> String {
        state.withLock { $0.argumentReads += 1 }
        return "fixture \(pid)"
    }
}
