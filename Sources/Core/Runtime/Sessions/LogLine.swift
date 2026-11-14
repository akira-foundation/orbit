import Foundation

public struct LogLine: Sendable, Equatable, Codable {
    public let timestamp: Date
    public let stream: LogStream
    public let text: String
    public let cursor: EventCursor?

    private enum CodingKeys: String, CodingKey {
        case timestamp, stream, text, cursor
    }

    public init(timestamp: Date, stream: LogStream, text: String, cursor: EventCursor? = nil) {
        self.timestamp = timestamp
        self.stream = stream
        self.text = text
        self.cursor = cursor
    }

    public init(from decoder: any Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        timestamp = try RFC3339Timestamp.decode(container, .timestamp)
        stream = try container.decode(LogStream.self, forKey: .stream)
        text = try container.decode(String.self, forKey: .text)
        cursor = try container.decodeIfPresent(EventCursor.self, forKey: .cursor)
    }

    public func encode(to encoder: any Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(RFC3339Timestamp.nanoseconds(timestamp), forKey: .timestamp)
        try container.encode(stream, forKey: .stream)
        try container.encode(text, forKey: .text)
        try container.encodeIfPresent(cursor, forKey: .cursor)
    }
}
