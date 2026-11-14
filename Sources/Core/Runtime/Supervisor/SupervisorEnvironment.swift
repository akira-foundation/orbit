enum SupervisorEnvironment {
    static let loaderPrefix = "DYLD_"
    static let deferredLoaderPrefix = "ORBIT_RUNTIME_CHILD_DYLD_"

    static func forSupervisor(_ base: EnvironmentList) -> EnvironmentList {
        EnvironmentList(
            base.entries.map { entry in
                entry.hasPrefix(loaderPrefix) ? deferredLoaderPrefix + entry.dropFirst(loaderPrefix.count) : entry
            })
    }

    static func forChild(_ own: EnvironmentList) -> EnvironmentList {
        EnvironmentList(
            own.entries.compactMap { entry in
                if entry.hasPrefix(loaderPrefix) { return nil }
                guard entry.hasPrefix(deferredLoaderPrefix) else { return entry }
                return loaderPrefix + entry.dropFirst(deferredLoaderPrefix.count)
            })
    }
}
