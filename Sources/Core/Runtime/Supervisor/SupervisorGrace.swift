public enum SupervisorGrace {
    public static let variable = "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS"
    public static let fallback: Duration = .seconds(5)

    public static func parse(_ value: String?) -> Duration {
        guard let milliseconds = value.flatMap({ Int64($0) }), milliseconds > 0 else { return fallback }
        return .milliseconds(milliseconds)
    }

    public static func environment(_ base: EnvironmentList, grace: Duration) -> EnvironmentList {
        var environment = base
        environment.remove(variable)
        environment.entries.append("\(variable)=\(milliseconds(grace > .zero ? grace : fallback))")
        return environment
    }

    private static func milliseconds(_ duration: Duration) -> Int64 {
        let (seconds, attoseconds) = duration.components
        return seconds * 1000 + attoseconds / 1_000_000_000_000_000
    }
}
