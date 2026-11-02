struct KnownRuntimeMembers: Sendable {
    private(set) var members: [Int32: ProcessIdentity]
    private(set) var unsettled: Set<Int32>

    init(supervisor: ProcessIdentity, inspector: any ProcessInspector) throws {
        let owned = try Self.owned(by: supervisor, inspector: inspector)
        members = Dictionary(uniqueKeysWithValues: owned.identities.map { ($0.pid, $0) })
        unsettled = Set(owned.unsettled)
    }

    var ordered: [ProcessIdentity] {
        members.values.sorted { $0.pid < $1.pid }
    }

    func containsOnly(_ pid: Int32) -> Bool {
        members.count == 1 && members[pid] != nil && unsettled.isEmpty
    }

    mutating func refresh(supervisor: ProcessIdentity, inspector: any ProcessInspector) throws {
        let owned = try Self.owned(by: supervisor, inspector: inspector)
        unsettled = Set(owned.unsettled)
        for identity in owned.identities {
            if let prior = members[identity.pid], prior != identity {
                throw RuntimeError.identityMismatch("PID \(identity.pid) changed while cleaning up")
            }
            members[identity.pid] = identity
        }
        for expected in ordered {
            do {
                guard try inspector.identity(of: expected.pid) == expected else {
                    throw RuntimeError.identityMismatch("PID \(expected.pid) changed while cleaning up")
                }
            } catch let error where error.contains(.processGone) {
                members[expected.pid] = nil
            }
        }
    }

    static func owned(by supervisor: ProcessIdentity, inspector: any ProcessInspector) throws -> ProcessListing {
        try inspector.verifyIdentity(supervisor)
        let group = try inspector.groupListing(supervisor.pgid)
        let descendants = try inspector.descendantListing(of: supervisor.pid)
        let combined = try combine(group.identities + descendants.identities)
        return ProcessListing(
            identities: combined.identities,
            unsettled: (group.unsettled + descendants.unsettled + combined.unsettled).filter { $0 != supervisor.pid }
        )
    }

    static func combine(_ identities: [ProcessIdentity]) throws -> ProcessListing {
        var combined: [Int32: ProcessIdentity] = [:]
        var conflicting: Set<Int32> = []
        for identity in identities {
            guard identity.pid > 0 else {
                throw RuntimeError.identityMismatch("invalid PID \(identity.pid)")
            }
            if let prior = combined[identity.pid], prior != identity {
                conflicting.insert(identity.pid)
            }
            combined[identity.pid] = identity
        }
        return ProcessListing(
            identities: combined.values.filter { !conflicting.contains($0.pid) }.sorted { $0.pid < $1.pid },
            unsettled: conflicting.sorted()
        )
    }
}
