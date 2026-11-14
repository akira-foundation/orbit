public struct DetectedPort: Sendable, Equatable {
    public let port: Int
    public let strong: Bool

    public init(port: Int, strong: Bool) {
        self.port = port
        self.strong = strong
    }
}
