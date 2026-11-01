public protocol ProcessInspector: Sendable {
    func identity(of pid: Int32) throws -> ProcessIdentity
    func groupListing(_ pgid: Int32) throws -> ProcessListing
    func descendantListing(of pid: Int32) throws -> ProcessListing
    func listenerListing(_ endpoint: Endpoint) throws -> ProcessListing
    func signal(_ expected: ProcessIdentity, _ signal: ProcessSignal) throws
}

extension ProcessInspector {
    public func groupMembers(_ pgid: Int32) throws -> [ProcessIdentity] {
        try groupListing(pgid).settled()
    }

    public func descendants(of pid: Int32) throws -> [ProcessIdentity] {
        try descendantListing(of: pid).settled()
    }

    public func listenerOwners(_ endpoint: Endpoint) throws -> [ProcessIdentity] {
        try listenerListing(endpoint).settled()
    }

    func verifyIdentity(_ expected: ProcessIdentity) throws {
        guard try identity(of: expected.pid) == expected else {
            throw RuntimeError.identityMismatch("PID \(expected.pid)")
        }
    }
}
