public struct SupervisorInvocation: Sendable, Equatable {
    public let sessionID: String
    public let token: String
    public let controlDescriptor: Int32
    public let directory: String
    public let command: [String]

    public init(sessionID: String, token: String, controlDescriptor: Int32, directory: String, command: [String]) {
        self.sessionID = sessionID
        self.token = token
        self.controlDescriptor = controlDescriptor
        self.directory = directory
        self.command = command
    }
}
