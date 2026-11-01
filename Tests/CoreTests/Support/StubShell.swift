import Core
import Synchronization

final class StubShell: LoginShellRunning {
    struct Call: Equatable {
        let shell: String
        let script: String
    }

    private let output: String?
    private let delay: Duration
    private let recorded = Mutex<[Call]>([])

    init(output: String?, delay: Duration = .zero) {
        self.output = output
        self.delay = delay
    }

    var calls: [Call] {
        recorded.withLock { $0 }
    }

    func run(shell: String, script: String, timeout: Duration) async -> String? {
        recorded.withLock { $0.append(Call(shell: shell, script: script)) }
        try? await Task.sleep(for: delay)
        return output
    }
}
