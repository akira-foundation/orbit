import Foundation

public struct SessionSnapshot: Sendable, Equatable, Codable {
    public var projectID: String
    public var status: ProjectStatus
    public var phase: String?
    public var pid: Int32
    public var port: Int
    public var startedAt: Date
    public var readyAt: Date?
    public var uptimeMs: Int64
    public var startupMs: Int64?
    public var attempts: Int?
    public var conns: Int
    public var lastActivity: Date
    public var error: String?

    private enum CodingKeys: String, CodingKey {
        case projectID = "projectId"
        case status, phase, pid, port, startedAt, readyAt, uptimeMs, startupMs, attempts, conns, lastActivity, error
    }

    public init(
        projectID: String, status: ProjectStatus, phase: String? = nil, pid: Int32, port: Int, startedAt: Date,
        readyAt: Date?, uptimeMs: Int64, startupMs: Int64?, attempts: Int? = nil, conns: Int, lastActivity: Date,
        error: String?
    ) {
        self.projectID = projectID
        self.status = status
        self.phase = phase
        self.pid = pid
        self.port = port
        self.startedAt = startedAt
        self.readyAt = readyAt
        self.uptimeMs = uptimeMs
        self.startupMs = startupMs
        self.attempts = attempts
        self.conns = conns
        self.lastActivity = lastActivity
        self.error = error
    }

    public init(from decoder: any Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        projectID = try container.decode(String.self, forKey: .projectID)
        status = try container.decode(ProjectStatus.self, forKey: .status)
        phase = try container.decodeIfPresent(String.self, forKey: .phase)
        pid = try container.decode(Int32.self, forKey: .pid)
        port = try container.decode(Int.self, forKey: .port)
        startedAt = try Self.date(container, .startedAt)
        readyAt = try container.contains(.readyAt) ? Self.date(container, .readyAt) : nil
        uptimeMs = try container.decode(Int64.self, forKey: .uptimeMs)
        startupMs = try container.decodeIfPresent(Int64.self, forKey: .startupMs)
        attempts = try container.decodeIfPresent(Int.self, forKey: .attempts)
        conns = try container.decode(Int.self, forKey: .conns)
        lastActivity = try Self.date(container, .lastActivity)
        error = try container.decodeIfPresent(String.self, forKey: .error)
    }

    public func encode(to encoder: any Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(projectID, forKey: .projectID)
        try container.encode(status, forKey: .status)
        try container.encodeIfPresent(phase, forKey: .phase)
        try container.encode(pid, forKey: .pid)
        try container.encode(port, forKey: .port)
        try container.encode(startedAt.formatted(.iso8601), forKey: .startedAt)
        try container.encodeIfPresent(readyAt?.formatted(.iso8601), forKey: .readyAt)
        try container.encode(uptimeMs, forKey: .uptimeMs)
        try container.encodeIfPresent(startupMs, forKey: .startupMs)
        try container.encodeIfPresent(attempts, forKey: .attempts)
        try container.encode(conns, forKey: .conns)
        try container.encode(lastActivity.formatted(.iso8601), forKey: .lastActivity)
        try container.encodeIfPresent(error, forKey: .error)
    }

    private static func date(_ container: KeyedDecodingContainer<CodingKeys>, _ key: CodingKeys) throws -> Date {
        let text = try container.decode(String.self, forKey: key)
        guard let date = try? Date(text, strategy: .iso8601) else {
            throw DecodingError.dataCorruptedError(
                forKey: key, in: container, debugDescription: "expected an RFC 3339 date, got \(text)")
        }
        return date
    }
}
