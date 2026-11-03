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
        var spawnedRecyclesInsideSignal = false
        var unsettledForListings = 0
        var conflictingForDescendantReads = 0
        var execsAfterTerm: String?
        var execsInsideSignal: String?
        var recyclesAfterTerm = false
        var unstableIdentityReads = 0
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
        var unsettled: Set<Int32> = []
        var unsettledListings: [Int32: Int] = [:]
        var conflictingDescendantReads: [Int32: Int] = [:]
        var confirmingListenerReadConflicts = false
        var groupReads = 0
        var groupFailure: (fromRead: Int, message: String)?
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

    var confirmingListenerReadConflicts: Bool {
        get { state.withLock { $0.confirmingListenerReadConflicts } }
        set { state.withLock { $0.confirmingListenerReadConflicts = newValue } }
    }

    func identity(of pid: Int32) throws -> ProcessIdentity {
        try state.withLock { state in
            if let reads = state.processes[pid]?.unstableIdentityReads, reads > 0 {
                state.processes[pid]?.unstableIdentityReads = reads - 1
                throw RuntimeError.identityMismatch("changed during enrichment")
            }
            if let remaining = state.pendingDeathReads[pid] {
                state.pendingDeathReads[pid] = remaining > 1 ? remaining - 1 : nil
                if remaining <= 1 {
                    state.alive.remove(pid)
                }
            }
            return try Self.identity(of: pid, in: state)
        }
    }

    var unsettled: Set<Int32> {
        get { state.withLock { $0.unsettled } }
        set { state.withLock { $0.unsettled = newValue } }
    }

    func failGroupListing(fromRead read: Int, message: String) {
        state.withLock { $0.groupFailure = (read, message) }
    }

    func groupListing(_ pgid: Int32) throws -> ProcessListing {
        try state.withLock { state in
            state.groupReads += 1
            if let failure = state.groupFailure, state.groupReads >= failure.fromRead {
                throw RuntimeError.message(failure.message)
            }
            return Self.listing(in: &state) { $0.pgid == pgid }
        }
    }

    func descendantListing(of pid: Int32) throws -> ProcessListing {
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
            let listed = Self.listing(in: &state) { $0.pid != pid && known.contains($0.pid) }
            return ProcessListing(
                identities: listed.identities.map { Self.descendantSnapshot(of: $0, in: &state) },
                unsettled: listed.unsettled
            )
        }
    }

    func listenerListing(_ endpoint: Endpoint) throws -> ProcessListing {
        try state.withLock { state in
            if let message = state.listenerError {
                throw RuntimeError.message(message)
            }
            guard !state.listenerReads.isEmpty else { return ProcessListing(identities: [], unsettled: []) }
            let pids = state.listenerReads.removeFirst()
            let conflicts = state.confirmingListenerReadConflicts && state.listenerReads.isEmpty
            return ProcessListing(
                identities: pids.filter { !state.unsettled.contains($0) }.compactMap { state.identities[$0] }.map {
                    conflicts ? Self.replacing($0, startID: "exec-\($0.pid)") : $0
                },
                unsettled: pids.filter { state.unsettled.contains($0) }
            )
        }
    }

    func signal(_ expected: ProcessIdentity, _ signal: ProcessSignal) throws {
        try state.withLock { state in
            let pid = expected.pid
            if state.processes[pid]?.recycleInsideSignal == true, let identity = state.identities[pid] {
                state.identities[pid] = Self.replacing(identity, startID: "recycled")
                state.processes[pid]?.recycleInsideSignal = false
            }
            if let image = state.processes[pid]?.execsInsideSignal, let identity = state.identities[pid] {
                state.identities[pid] = Self.execing(identity, image: image)
                state.processes[pid]?.execsInsideSignal = nil
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
                    pid: child, ignoresTerm: true, denySignalOnce: state.processes[pid]?.spawnedDeniesSignalOnce,
                    recycleInsideSignal: state.processes[pid]?.spawnedRecyclesInsideSignal == true
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
            if signal == .terminate, let image = state.processes[pid]?.execsAfterTerm {
                state.identities[pid] = Self.execing(expected, image: image)
                state.processes[pid]?.execsAfterTerm = nil
            }
            if signal == .terminate, state.processes[pid]?.recyclesAfterTerm == true {
                state.identities[pid] = Self.replacing(expected, startID: "recycled")
                state.processes[pid]?.recyclesAfterTerm = false
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
        if process.unsettledForListings > 0 {
            state.unsettledListings[process.pid] = process.unsettledForListings
        }
        if process.conflictingForDescendantReads > 0 {
            state.conflictingDescendantReads[process.pid] = process.conflictingForDescendantReads
        }
    }

    private static func identity(of pid: Int32, in state: State) throws -> ProcessIdentity {
        guard state.alive.contains(pid), let identity = state.identities[pid] else {
            throw RuntimeError.processGone
        }
        return identity
    }

    private static func listing(in state: inout State, where include: (ProcessIdentity) -> Bool) -> ProcessListing {
        let matching = state.alive.compactMap { state.identities[$0] }.filter(include).sorted { $0.pid < $1.pid }
        let unsettled = matching.map(\.pid).filter {
            state.unsettled.contains($0) || state.unsettledListings[$0] != nil
        }
        for pid in unsettled {
            guard let remaining = state.unsettledListings[pid] else { continue }
            state.unsettledListings[pid] = remaining > 1 ? remaining - 1 : nil
        }
        return ProcessListing(
            identities: matching.filter { !unsettled.contains($0.pid) },
            unsettled: unsettled
        )
    }

    private static func descendantSnapshot(of identity: ProcessIdentity, in state: inout State) -> ProcessIdentity {
        guard let remaining = state.conflictingDescendantReads[identity.pid] else { return identity }
        state.conflictingDescendantReads[identity.pid] = remaining > 1 ? remaining - 1 : nil
        return replacing(identity, startID: "exec-\(identity.pid)")
    }

    private static func execing(_ identity: ProcessIdentity, image: String) -> ProcessIdentity {
        ProcessIdentity(
            pid: identity.pid, pgid: identity.pgid, startID: identity.startID, executable: image,
            arguments: "\(image) \(identity.pid)")
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
