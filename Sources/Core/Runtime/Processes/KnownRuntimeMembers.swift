struct KnownRuntimeMembers: Sendable {
    private(set) var members: [Int32: ProcessIdentity]

    init(supervisor: ProcessIdentity, inspector: any ProcessInspector) throws {
        let owned = try Self.owned(by: supervisor, inspector: inspector)
        members = Dictionary(uniqueKeysWithValues: owned.map { ($0.pid, $0) })
    }

    var ordered: [ProcessIdentity] {
        members.values.sorted { $0.pid < $1.pid }
    }

    func containsOnly(_ pid: Int32) -> Bool {
        members.count == 1 && members[pid] != nil
    }

    mutating func refresh(supervisor: ProcessIdentity, inspector: any ProcessInspector) throws {
        for identity in try Self.owned(by: supervisor, inspector: inspector) {
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

    static func owned(by supervisor: ProcessIdentity, inspector: any ProcessInspector) throws -> [ProcessIdentity] {
        try inspector.verifyIdentity(supervisor)
        let group = try inspector.groupMembers(supervisor.pgid)
        let descendants = try inspector.descendants(of: supervisor.pid)
        return try combine(group + descendants)
    }

    static func combine(_ identities: [ProcessIdentity]) throws -> [ProcessIdentity] {
        var combined: [Int32: ProcessIdentity] = [:]
        for identity in identities {
            guard identity.pid > 0 else {
                throw RuntimeError.identityMismatch("invalid PID \(identity.pid)")
            }
            if let prior = combined[identity.pid], prior != identity {
                throw RuntimeError.identityMismatch("conflicting snapshots for PID \(identity.pid)")
            }
            combined[identity.pid] = identity
        }
        return combined.values.sorted { $0.pid < $1.pid }
    }
}
