public protocol LoginShellRunning: Sendable {
    func run(shell: String, script: String, timeout: Duration) async -> String?
}
