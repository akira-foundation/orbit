public struct EventCursor: Sendable, Equatable, Codable {
    public let timestamp: Int64
    public let id: Int64

    public init(timestamp: Int64, id: Int64) {
        self.timestamp = timestamp
        self.id = id
    }
}
