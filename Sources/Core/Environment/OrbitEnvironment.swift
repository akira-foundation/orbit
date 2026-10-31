public enum OrbitEnvironment: String, Sendable, CaseIterable {
    case production
    case development

    public static func current(bundleIdentifier: String?) -> OrbitEnvironment {
        bundleIdentifier == OrbitEnvironment.development.bundleIdentifier ? .development : .production
    }

    public var bundleIdentifier: String {
        switch self {
        case .production: "io.akira.orbit"
        case .development: "io.akira.orbit.dev"
        }
    }

    public var domainSuffix: String {
        switch self {
        case .production: "orbit.test"
        case .development: "orbit-dev.test"
        }
    }

    public var loopbackAddress: String {
        switch self {
        case .production: "127.0.0.2"
        case .development: "127.0.0.3"
        }
    }

    public var helperMachServiceName: String {
        "\(bundleIdentifier).helper"
    }

    public var agentMachServiceName: String {
        "\(bundleIdentifier).agent"
    }

    public var dataDirectoryName: String {
        switch self {
        case .production: "Orbit"
        case .development: "Orbit (Dev)"
        }
    }
}
