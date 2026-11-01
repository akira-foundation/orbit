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

    init(_ reads: [Read]) {
        self.reads = Mutex(reads)
    }

    func identity(of pid: Int32) throws -> ProcessIdentity {
        let read = reads.withLock { $0.count > 1 ? $0.removeFirst() : $0[0] }
        switch read {
        case .changed: throw RuntimeError.identityMismatch("changed during enrichment")
        case .parentGroup: return Self.identity(pid, pgid: getpgrp())
        case .leader: return Self.identity(pid, pgid: pid)
        }
    }

    func groupListing(_ pgid: Int32) throws -> ProcessListing {
        ProcessListing(identities: [], unsettled: [])
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
