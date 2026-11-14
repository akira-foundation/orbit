public protocol SupervisorLaunching: Sendable {
    func launch(_ spec: SupervisorSpec) async throws -> any SupervisorHandle
}
