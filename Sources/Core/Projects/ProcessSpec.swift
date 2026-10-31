public struct ProcessSpec: Sendable, Equatable {
    public var role: ProcessRole
    public var kind: RuntimeKind
    public var workDirectory: String?
    public var command: String?
    public var port: Int?
    public var phpVersion: String?

    public init(
        role: ProcessRole,
        kind: RuntimeKind,
        workDirectory: String? = nil,
        command: String? = nil,
        port: Int? = nil,
        phpVersion: String? = nil
    ) {
        self.role = role
        self.kind = kind
        self.workDirectory = workDirectory
        self.command = command
        self.port = port
        self.phpVersion = phpVersion
    }
}
