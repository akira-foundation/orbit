import Foundation

public actor LoginPathResolver {
    public static let marker = "__ORBIT_PATH__:"
    public static let timeout: Duration = .seconds(5)

    private let runner: any LoginShellRunning
    private let shell: String
    private let fallback: LoginPathFallback
    private var pending: Task<String, Never>?

    public init(runner: any LoginShellRunning, shell: String?, home: String?, fileManager: FileManager = .default) {
        self.runner = runner
        self.shell = shell.flatMap { $0.isEmpty ? nil : $0 } ?? "/bin/zsh"
        self.fallback = LoginPathFallback(home: home, fileManager: fileManager)
    }

    public func path() async -> String {
        if let pending {
            return await pending.value
        }
        let task = Task { await self.queryShell() ?? self.fallback.path() }
        pending = task
        return await task.value
    }

    public func apply(to environment: EnvironmentList) async -> EnvironmentList {
        var result = environment
        let login = await path()
        if let existing = environment.value(of: "PATH") {
            result.set("PATH", PathList.merge(existing, login))
        } else {
            result.set("PATH", login)
        }
        return result
    }

    private func queryShell() async -> String? {
        let script = "echo \(Self.marker)$PATH"
        guard let output = await runner.run(shell: shell, script: script, timeout: Self.timeout) else {
            return nil
        }
        for line in output.split(whereSeparator: \.isNewline) where line.hasPrefix(Self.marker) {
            let value = line.dropFirst(Self.marker.count).trimmingCharacters(in: .whitespaces)
            if !value.isEmpty {
                return value
            }
        }
        return nil
    }
}
