import Foundation

public struct LogLine: Sendable, Equatable, Codable {
    public let timestamp: Date
    public let stream: LogStream
    public let text: String

    public init(timestamp: Date, stream: LogStream, text: String) {
        self.timestamp = timestamp
        self.stream = stream
        self.text = text
    }
}
