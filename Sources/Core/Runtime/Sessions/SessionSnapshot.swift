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
        startedAt = try RFC3339Timestamp.decode(container, .startedAt)
        readyAt =
            try container.contains(.readyAt) && !container.decodeNil(forKey: .readyAt)
            ? RFC3339Timestamp.decode(container, .readyAt) : nil
        uptimeMs = try container.decode(Int64.self, forKey: .uptimeMs)
        startupMs = try container.decodeIfPresent(Int64.self, forKey: .startupMs)
        attempts = try container.decodeIfPresent(Int.self, forKey: .attempts)
        conns = try container.decode(Int.self, forKey: .conns)
        lastActivity = try RFC3339Timestamp.decode(container, .lastActivity)
        error = try container.decodeIfPresent(String.self, forKey: .error)
    }

    public func encode(to encoder: any Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(projectID, forKey: .projectID)
        try container.encode(status, forKey: .status)
        try container.encodeIfPresent(phase.flatMap { $0.isEmpty ? nil : $0 }, forKey: .phase)
        try container.encode(pid, forKey: .pid)
        try container.encode(port, forKey: .port)
        try container.encode(RFC3339Timestamp.seconds(startedAt), forKey: .startedAt)
        try container.encodeIfPresent(readyAt.map(RFC3339Timestamp.seconds), forKey: .readyAt)
        try container.encode(uptimeMs, forKey: .uptimeMs)
        try container.encodeIfPresent(startupMs.flatMap { $0 == 0 ? nil : $0 }, forKey: .startupMs)
        try container.encodeIfPresent(attempts.flatMap { $0 == 0 ? nil : $0 }, forKey: .attempts)
        try container.encode(conns, forKey: .conns)
        try container.encode(RFC3339Timestamp.seconds(lastActivity), forKey: .lastActivity)
        try container.encodeIfPresent(error.flatMap { $0.isEmpty ? nil : $0 }, forKey: .error)
    }
}
