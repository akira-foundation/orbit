public struct ProcessListing: Sendable, Equatable {
    public let identities: [ProcessIdentity]
    public let unsettled: [Int32]

    public init(identities: [ProcessIdentity], unsettled: [Int32]) {
        self.identities = identities
        self.unsettled = unsettled
    }

    func settled() throws -> [ProcessIdentity] {
        guard let pid = unsettled.first else { return identities }
        throw RuntimeError.identityMismatch("PID \(pid) did not settle")
    }
}
