public struct Endpoint: Sendable, Hashable {
    public let host: String
    public let port: Int

    public init(host: String, port: Int) {
        self.host = host
        self.port = port
    }
}
