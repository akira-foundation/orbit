import Foundation

public struct Project: Sendable, Equatable, Identifiable {
    public var id: String
    public var name: String
    public var path: String
    public var slug: String
    public var localDomain: String
    public var framework: String?
    public var packageManager: String?
    public var devCommand: String?
    public var devPort: Int?
    public var runtimeKind: RuntimeKind
    public var nodeVersion: String?
    public var phpVersion: String?
    public var pythonVersion: String?
    public var status: ProjectStatus
    public var secure: Bool
    public var pinned: Bool
    public var installedHash: String?
    public var parkedFolder: String?
    public var createdAt: Date
    public var updatedAt: Date
    public var scripts: [ProjectScript]
    public var processes: [ProcessSpec]

    public init(
        id: String,
        name: String,
        path: String,
        slug: String,
        localDomain: String,
        framework: String? = nil,
        packageManager: String? = nil,
        devCommand: String? = nil,
        devPort: Int? = nil,
        runtimeKind: RuntimeKind = .command,
        nodeVersion: String? = nil,
        phpVersion: String? = nil,
        pythonVersion: String? = nil,
        status: ProjectStatus = .stopped,
        secure: Bool = false,
        pinned: Bool = false,
        installedHash: String? = nil,
        parkedFolder: String? = nil,
        createdAt: Date = .now,
        updatedAt: Date = .now,
        scripts: [ProjectScript] = [],
        processes: [ProcessSpec] = []
    ) {
        self.id = id
        self.name = name
        self.path = path
        self.slug = slug
        self.localDomain = localDomain
        self.framework = framework
        self.packageManager = packageManager
        self.devCommand = devCommand
        self.devPort = devPort
        self.runtimeKind = runtimeKind
        self.nodeVersion = nodeVersion
        self.phpVersion = phpVersion
        self.pythonVersion = pythonVersion
        self.status = status
        self.secure = secure
        self.pinned = pinned
        self.installedHash = installedHash
        self.parkedFolder = parkedFolder
        self.createdAt = createdAt
        self.updatedAt = updatedAt
        self.scripts = scripts
        self.processes = processes
    }

    public init(
        analysis: ProjectAnalysis,
        slug: String,
        localDomain: String,
        parkedFolder: String? = nil,
        now: Date = .now
    ) {
        let processes =
            analysis.processes.isEmpty
            ? [
                ProcessSpec(
                    role: .backend,
                    kind: analysis.runtimeKind,
                    command: analysis.devCommand,
                    port: analysis.devPort,
                    phpVersion: analysis.phpVersion)
            ]
            : analysis.processes
        self.init(
            id: UUID().uuidString.lowercased(),
            name: analysis.name,
            path: analysis.path,
            slug: slug,
            localDomain: localDomain,
            framework: analysis.framework,
            packageManager: analysis.packageManager,
            devCommand: analysis.devCommand,
            devPort: analysis.devPort,
            runtimeKind: analysis.runtimeKind,
            nodeVersion: analysis.nodeVersion,
            phpVersion: analysis.phpVersion,
            pythonVersion: analysis.pythonVersion,
            parkedFolder: parkedFolder,
            createdAt: now,
            updatedAt: now,
            scripts: analysis.scripts.sorted { $0.key < $1.key }.map { ProjectScript(name: $0.key, command: $0.value) },
            processes: processes)
    }
}
