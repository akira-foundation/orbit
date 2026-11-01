import Synchronization

@testable import Core

final class FakeProcessInspector: ProcessInspector {
    struct Process: Sendable {
        var pid: Int32
        var pgid: Int32?
        var ppid: Int32?
        var startID: String?
        var ignoresTerm = false
        var denySignalOnce: String?
        var recycleInsideSignal = false
        var changeGroupAfterTerm: Int32?
        var exitBeforeSignal = false
        var survivesKillForReads = 0
        var spawnsOnTerm: Int32?
        var spawnsOnKill: Int32?
        var spawnedDeniesSignalOnce: String?
    }

    private struct State {
        var identities: [Int32: ProcessIdentity] = [:]
        var parents: [Int32: Int32] = [:]
        var processes: [Int32: Process] = [:]
        var alive: Set<Int32> = []
        var signals: [(Int32, ProcessSignal)] = []
        var listenerReads: [[Int32]] = []
        var listenerError: String?
        var pendingDeathReads: [Int32: Int] = [:]
        var signalVerifiesIdentity = true
    }

    private let state = Mutex(State())
    let supervisorPID: Int32

    init(supervisorPID: Int32) {
        self.supervisorPID = supervisorPID
    }

    static func supervised(supervisor: Int32 = 4100, members: [Process] = []) -> FakeProcessInspector {
        let fake = FakeProcessInspector(supervisorPID: supervisor)
        fake.add(Process(pid: supervisor, pgid: supervisor, ppid: 0))
        for member in members {
            fake.add(member)
        }
        return fake
    }

    func add(_ process: Process) {
        state.withLock { Self.insert(process, supervisorPID: supervisorPID, into: &$0) }
    }

    func identity(_ pid: Int32) -> ProcessIdentity {
        state.withLock {
            $0.identities[pid] ?? ProcessIdentity(pid: pid, pgid: 0, startID: "", executable: "", arguments: "")
        }
    }

    func setStartID(_ pid: Int32, _ startID: String) {
        state.withLock { state in
            guard let identity = state.identities[pid] else { return }
            state.identities[pid] = Self.replacing(identity, startID: startID)
        }
    }

    func isAlive(_ pid: Int32) -> Bool {
        state.withLock { $0.alive.contains(pid) }
    }

    var signals: [(Int32, ProcessSignal)] {
        state.withLock { $0.signals }
    }

    var listenerReads: [[Int32]] {
        get { state.withLock { $0.listenerReads } }
        set { state.withLock { $0.listenerReads = newValue } }
    }

    var listenerError: String? {
        get { state.withLock { $0.listenerError } }
        set { state.withLock { $0.listenerError = newValue } }
    }

    var signalVerifiesIdentity: Bool {
        get { state.withLock { $0.signalVerifiesIdentity } }
        set { state.withLock { $0.signalVerifiesIdentity = newValue } }
    }

    func identity(of pid: Int32) throws -> ProcessIdentity {
        try state.withLock { state in
            if let remaining = state.pendingDeathReads[pid] {
                state.pendingDeathReads[pid] = remaining > 1 ? remaining - 1 : nil
                if remaining <= 1 {
                    state.alive.remove(pid)
                }
            }
            return try Self.identity(of: pid, in: state)
        }
    }

    func groupMembers(_ pgid: Int32) throws -> [ProcessIdentity] {
        state.withLock { state in Self.members(in: state) { $0.pgid == pgid } }
    }

    func descendants(of pid: Int32) throws -> [ProcessIdentity] {
        state.withLock { state in
            var known: Set<Int32> = [pid]
            var changed = true
            while changed {
                changed = false
                for (child, parent) in state.parents
                where state.alive.contains(child) && known.contains(parent) && !known.contains(child) {
                    known.insert(child)
                    changed = true
                }
            }
            return Self.members(in: state) { $0.pid != pid && known.contains($0.pid) }
        }
    }

    func listenerOwners(_ endpoint: Endpoint) throws -> [ProcessIdentity] {
        try state.withLock { state in
            if let message = state.listenerError {
                throw RuntimeError.message(message)
            }
            guard !state.listenerReads.isEmpty else { return [] }
            return state.listenerReads.removeFirst().compactMap { state.identities[$0] }
        }
    }

    func signal(_ expected: ProcessIdentity, _ signal: ProcessSignal) throws {
        try state.withLock { state in
            let pid = expected.pid
            if state.processes[pid]?.recycleInsideSignal == true, let identity = state.identities[pid] {
                state.identities[pid] = Self.replacing(identity, startID: "recycled")
                state.processes[pid]?.recycleInsideSignal = false
            }
            if state.processes[pid]?.exitBeforeSignal == true {
                state.alive.remove(pid)
                state.processes[pid]?.exitBeforeSignal = false
            }
            let actual = try Self.identity(of: pid, in: state)
            guard !state.signalVerifiesIdentity || actual == expected else {
                throw RuntimeError.identityMismatch("PID \(pid)")
            }
            if let message = state.processes[pid]?.denySignalOnce {
                state.processes[pid]?.denySignalOnce = nil
                throw RuntimeError.message(message)
            }
            state.signals.append((pid, signal))
            if signal == .terminate, let child = state.processes[pid]?.spawnsOnTerm {
                let spawned = Process(
                    pid: child, ignoresTerm: true, denySignalOnce: state.processes[pid]?.spawnedDeniesSignalOnce
                )
                Self.insert(spawned, supervisorPID: supervisorPID, into: &state)
            }
            if signal == .kill, let child = state.processes[pid]?.spawnsOnKill {
                Self.insert(Process(pid: child), supervisorPID: supervisorPID, into: &state)
            }
            if signal == .terminate, let group = state.processes[pid]?.changeGroupAfterTerm {
                state.identities[pid] = Self.replacing(expected, pgid: group)
                state.parents[pid] = 0
                state.processes[pid]?.changeGroupAfterTerm = nil
            }
            if signal == .kill, let reads = state.processes[pid]?.survivesKillForReads, reads > 0 {
                state.pendingDeathReads[pid] = reads
                return
            }
            if signal == .kill || state.processes[pid]?.ignoresTerm != true {
                state.alive.remove(pid)
            }
        }
    }

    private static func insert(_ process: Process, supervisorPID: Int32, into state: inout State) {
        let pgid = process.pgid ?? supervisorPID
        state.identities[process.pid] = ProcessIdentity(
            pid: process.pid,
            pgid: pgid,
            startID: process.startID ?? "start-\(process.pid)",
            executable: "/fixture/process",
            arguments: "fixture \(process.pid)"
        )
        state.parents[process.pid] = process.ppid ?? (process.pid == pgid ? 0 : pgid)
        state.processes[process.pid] = process
        state.alive.insert(process.pid)
    }

    private static func identity(of pid: Int32, in state: State) throws -> ProcessIdentity {
        guard state.alive.contains(pid), let identity = state.identities[pid] else {
            throw RuntimeError.processGone
        }
        return identity
    }

    private static func members(in state: State, where include: (ProcessIdentity) -> Bool) -> [ProcessIdentity] {
        state.alive.compactMap { state.identities[$0] }.filter(include).sorted { $0.pid < $1.pid }
    }

    private static func replacing(
        _ identity: ProcessIdentity, pgid: Int32? = nil, startID: String? = nil
    ) -> ProcessIdentity {
        ProcessIdentity(
            pid: identity.pid,
            pgid: pgid ?? identity.pgid,
            startID: startID ?? identity.startID,
            executable: identity.executable,
            arguments: identity.arguments
        )
    }
}
