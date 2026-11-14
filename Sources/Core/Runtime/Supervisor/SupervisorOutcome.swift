import Dispatch
import Synchronization

final class SupervisorOutcome: Sendable {
    private let error = Mutex<(any Error)?>(nil)
    private let finished = DispatchSemaphore(value: 0)

    func finish(_ failure: (any Error)?) {
        error.withLock { $0 = failure }
        finished.signal()
    }

    func wait() -> (any Error)? {
        finished.wait()
        return error.withLock { $0 }
    }
}
