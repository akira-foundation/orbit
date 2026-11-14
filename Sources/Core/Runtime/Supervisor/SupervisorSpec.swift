public struct SupervisorSpec: Sendable {
    public var sessionID: String
    public var token: String
    public var directory: String
    public var command: [String]
    public var environment: EnvironmentList?
    public var grace: Duration

    public init(
        sessionID: String, token: String, directory: String, command: [String], environment: EnvironmentList? = nil,
        grace: Duration = SupervisorGrace.fallback
    ) {
        self.sessionID = sessionID
        self.token = token
        self.directory = directory
        self.command = command
        self.environment = environment
        self.grace = grace
    }
}
