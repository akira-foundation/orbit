import Core
import Synchronization

final class StubShell: LoginShellRunning {
    struct Call: Equatable {
        let shell: String
        let script: String
    }

    private let output: String?
    private let recorded = Mutex<[Call]>([])

    init(output: String?) {
        self.output = output
    }

    var calls: [Call] {
        recorded.withLock { $0 }
    }

    func run(shell: String, script: String, timeout: Duration) async -> String? {
        recorded.withLock { $0.append(Call(shell: shell, script: script)) }
        return output
    }
}
