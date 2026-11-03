import Foundation

public actor RuntimeManager {
    public static let cooldown: Duration = .seconds(10)
    public static let readinessTimeout: Duration = .seconds(60)
    public static let stopGrace: Duration = .seconds(5)
    public static let restartPause: Duration = .milliseconds(300)
    public static let stopAllTimeout: Duration = .seconds(3)
    public static let stopPoll: Duration = .milliseconds(50)
    static let overlongLineWarning = "output line longer than 1 MiB discarded"
    static let servicesTimeout: Duration = .seconds(60)

    let store: Store
    let launcher: any SupervisorLaunching
    let inspector: any ProcessInspector
    let endpoints: any EndpointReserving
    let services: any ServiceCoordinating
    let loginPath: LoginPathResolver
    let events: EventLog
    let sink: any RuntimeEventSink
    let clock: any RuntimeClock
    let baseEnvironment: EnvironmentList
    var sessions: [String: RuntimeSession] = [:]
    let statusWrites = StatusWrites()
    var supervising: [String: RuntimeSession] = [:]
    var launching: Set<String> = []
    var abandonedLaunches: Set<String> = []
    var closing = false
    var watchdogs: [String: Task<Void, Never>] = [:]
    var restarts: [String: Task<Void, Never>] = [:]
    var attempts: [String: Int] = [:]
    var connections: [String: Int] = [:]

    public init(
        store: Store, launcher: any SupervisorLaunching, inspector: any ProcessInspector,
        endpoints: any EndpointReserving, services: any ServiceCoordinating = NoServices(),
        loginPath: LoginPathResolver, events: EventLog, sink: any RuntimeEventSink,
        clock: any RuntimeClock = SystemClock(), baseEnvironment: EnvironmentList = .process()
    ) {
        self.store = store
        self.launcher = launcher
        self.inspector = inspector
        self.endpoints = endpoints
        self.services = services
        self.loginPath = loginPath
        self.events = events
        self.sink = sink
        self.clock = clock
        self.baseEnvironment = baseEnvironment
    }

    public func status(_ projectID: String) -> SessionSnapshot {
        guard let session = sessions[projectID] else {
            let never = Date(timeIntervalSince1970: 0)
            return SessionSnapshot(
                projectID: projectID, status: .stopped, pid: 0, port: 0, startedAt: never, readyAt: nil, uptimeMs: 0,
                startupMs: nil, conns: 0, lastActivity: never, error: nil)
        }
        return session.snapshot(conns: connections[projectID] ?? 0, attempts: attempts[projectID])
    }

    public func logs(_ projectID: String) -> [LogLine] {
        sessions[projectID]?.logs() ?? []
    }

    public func port(_ projectID: String) -> Int {
        sessions[projectID]?.endpoint?.port ?? 0
    }

    public func isRunning(_ projectID: String) -> Bool {
        guard let status = sessions[projectID]?.status else { return false }
        return status.isLive
    }

    public func connectionOpened(_ projectID: String) {
        guard !projectID.isEmpty else { return }
        connections[projectID, default: 0] += 1
    }

    public func connectionClosed(_ projectID: String) {
        guard !projectID.isEmpty else { return }
        if let count = connections[projectID], count > 1 {
            connections[projectID] = count - 1
        } else {
            connections[projectID] = nil
        }
    }

    public func history(_ projectID: String, since: Int64, limit: Int) async throws -> [LogLine] {
        await events.flush()
        return try await events.history(projectID: projectID, since: since, limit: limit)
    }

    public func history(_ projectID: String, after cursor: EventCursor, limit: Int) async throws -> [LogLine] {
        await events.flush()
        return try await events.history(projectID: projectID, after: cursor, limit: limit)
    }

    public func record(projectID: String?, level: EventLevel, source: EventSource, text: String) {
        events.push(EventRecord(projectID: projectID, level: level, source: source, text: text))
    }

    func isCurrent(_ session: RuntimeSession) -> Bool {
        sessions[session.projectID] === session
    }

    func publish(_ name: RuntimeEventName, _ session: RuntimeSession) {
        sink.emit(
            .status(
                name,
                session.snapshot(conns: connections[session.projectID] ?? 0, attempts: attempts[session.projectID])))
    }

    func persist(_ status: ProjectStatus, _ session: RuntimeSession) async {
        guard isCurrent(session) else { return }
        await persist(status, projectID: session.projectID)
    }

    func persist(_ status: ProjectStatus, projectID: String) async {
        let ticket = statusWrites.ticket(for: projectID)
        await statusWrites.write(status, projectID: projectID, ticket: ticket, to: store)
    }

    func log(_ received: LogLine, _ session: RuntimeSession) {
        let line = LogLine(
            timestamp: received.timestamp, stream: received.stream, text: StoredText.limited(received.text))
        session.addLog(line)
        events.push(
            EventRecord(
                projectID: session.projectID, sessionID: session.sessionID,
                level: line.stream == .stderr ? .warn : .info, source: .runtime, stream: line.stream,
                text: line.text))
        if isCurrent(session) {
            sink.emit(.log(projectID: session.projectID, line))
        }
    }
}
