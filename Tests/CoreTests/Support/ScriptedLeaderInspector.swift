import Darwin
import Synchronization

@testable import Core

final class ScriptedLeaderInspector: ProcessInspector {
    enum Read: Sendable {
        case parentGroup
        case changed
        case leader
    }

    private let reads: Mutex<[Read]>
    private let inspected = Mutex<Set<Int32>>([])
    private let group: ProcessListing

    init(_ reads: [Read], group: ProcessListing = ProcessListing(identities: [], unsettled: [])) {
        self.reads = Mutex(reads)
        self.group = group
    }

    var inspectedPIDs: Set<Int32> {
        inspected.withLock { $0 }
    }

    func identity(of pid: Int32) throws -> ProcessIdentity {
        inspected.withLock { _ = $0.insert(pid) }
        let read = reads.withLock { $0.count > 1 ? $0.removeFirst() : $0[0] }
        switch read {
        case .changed: throw RuntimeError.identityMismatch("changed during enrichment")
        case .parentGroup: return Self.identity(pid, pgid: getpgrp())
        case .leader: return Self.identity(pid, pgid: pid)
        }
    }

    func groupListing(_ pgid: Int32) throws -> ProcessListing {
        group
    }

    func descendantListing(of pid: Int32) throws -> ProcessListing {
        ProcessListing(identities: [], unsettled: [])
    }

    func listenerListing(_ endpoint: Endpoint) throws -> ProcessListing {
        ProcessListing(identities: [], unsettled: [])
    }

    func signal(_ expected: ProcessIdentity, _ signal: ProcessSignal) throws {}

    private static func identity(_ pid: Int32, pgid: Int32) -> ProcessIdentity {
        ProcessIdentity(pid: pid, pgid: pgid, startID: "start-\(pid)", executable: "fixture", arguments: "fixture")
    }
}
