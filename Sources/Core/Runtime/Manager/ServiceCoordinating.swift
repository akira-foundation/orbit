public protocol ServiceCoordinating: Sendable {
    func projectStarting(id: String, path: String, slug: String) async throws -> [String: String]
    func projectStopped(id: String) async
}
