public struct EventRecord: Sendable, Equatable {
    public var timestamp: Int64
    public var projectID: String?
    public var sessionID: String?
    public var level: EventLevel
    public var source: EventSource
    public var stream: LogStream?
    public var text: String

    public init(
        timestamp: Int64 = 0, projectID: String? = nil, sessionID: String? = nil, level: EventLevel = .info,
        source: EventSource = .app, stream: LogStream? = nil, text: String
    ) {
        self.timestamp = timestamp
        self.projectID = projectID
        self.sessionID = sessionID
        self.level = level
        self.source = source
        self.stream = stream
        self.text = text
    }
}
