public struct ProjectAnalysis: Sendable, Equatable {
    public var name: String
    public var path: String
    public var packageManager: String?
    public var framework: String?
    public var devCommand: String?
    public var devPort: Int?
    public var runtimeKind: RuntimeKind
    public var nodeVersion: String?
    public var phpVersion: String?
    public var pythonVersion: String?
    public var scripts: [String: String]
    public var processes: [ProcessSpec]
    public var ambiguousReason: String?

    public var isAmbiguous: Bool {
        ambiguousReason != nil
    }

    public init(
        name: String,
        path: String,
        packageManager: String? = nil,
        framework: String? = nil,
        devCommand: String? = nil,
        devPort: Int? = nil,
        runtimeKind: RuntimeKind = .command,
        nodeVersion: String? = nil,
        phpVersion: String? = nil,
        pythonVersion: String? = nil,
        scripts: [String: String] = [:],
        processes: [ProcessSpec] = [],
        ambiguousReason: String? = nil
    ) {
        self.name = name
        self.path = path
        self.packageManager = packageManager
        self.framework = framework
        self.devCommand = devCommand
        self.devPort = devPort
        self.runtimeKind = runtimeKind
        self.nodeVersion = nodeVersion
        self.phpVersion = phpVersion
        self.pythonVersion = pythonVersion
        self.scripts = scripts
        self.processes = processes
        self.ambiguousReason = ambiguousReason
    }
}
