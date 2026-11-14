public struct NoServices: ServiceCoordinating {
    public init() {}

    public func projectStarting(id: String, path: String, slug: String) async throws -> [String: String] {
        [:]
    }

    public func projectStopped(id: String) async {}
}
