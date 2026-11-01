public protocol ProcessInspector: Sendable {
    func identity(of pid: Int32) throws -> ProcessIdentity
    func groupMembers(_ pgid: Int32) throws -> [ProcessIdentity]
    func descendants(of pid: Int32) throws -> [ProcessIdentity]
    func listenerOwners(_ endpoint: Endpoint) throws -> [ProcessIdentity]
    func signal(_ expected: ProcessIdentity, _ signal: ProcessSignal) throws
}

extension ProcessInspector {
    func verifyIdentity(_ expected: ProcessIdentity) throws {
        guard try identity(of: expected.pid) == expected else {
            throw RuntimeError.identityMismatch("PID \(expected.pid)")
        }
    }
}
