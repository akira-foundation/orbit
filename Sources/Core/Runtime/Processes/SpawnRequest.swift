public struct SpawnRequest: Sendable {
    public var executable: String
    public var arguments: [String]
    public var environment: EnvironmentList
    public var directory: String?
    public var group: ProcessGroupPlacement
    public var captureOutput: Bool
    public var controlDescriptor: Int32?
    public var standardInput: Int32?

    public init(
        executable: String, arguments: [String] = [], environment: EnvironmentList, directory: String? = nil,
        group: ProcessGroupPlacement = .newLeader, captureOutput: Bool = false, controlDescriptor: Int32? = nil,
        standardInput: Int32? = nil
    ) {
        self.executable = executable
        self.arguments = arguments
        self.environment = environment
        self.directory = directory
        self.group = group
        self.captureOutput = captureOutput
        self.controlDescriptor = controlDescriptor
        self.standardInput = standardInput
    }
}
