public struct HTTPEndpoint: Sendable, Equatable {
    public let scheme: String
    public let host: String
    public let port: Int

    public init(scheme: String, host: String, port: Int) {
        self.scheme = scheme
        self.host = host
        self.port = port
    }
}
