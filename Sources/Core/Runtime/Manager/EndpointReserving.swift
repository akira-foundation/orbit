public protocol EndpointReserving: Sendable {
    func reserve() throws -> Endpoint
}
