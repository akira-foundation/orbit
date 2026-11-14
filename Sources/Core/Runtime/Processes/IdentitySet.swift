public enum IdentitySet {
    public static func same(_ lhs: [ProcessIdentity], _ rhs: [ProcessIdentity]) -> Bool {
        guard lhs.count == rhs.count else { return false }
        var remaining: [Int32: ProcessIdentity] = [:]
        for identity in lhs {
            guard identity.pid > 0, remaining[identity.pid] == nil else { return false }
            remaining[identity.pid] = identity
        }
        for identity in rhs {
            guard remaining.removeValue(forKey: identity.pid) == identity else { return false }
        }
        return remaining.isEmpty
    }
}
