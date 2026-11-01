import Darwin
import Foundation
import Synchronization

final class ProcessReaper: Sendable {
    private let state = Mutex(ReapState.running([]))

    init(pid: Int32) {
        let thread = Thread { [self] in
            finish(with: Self.wait(for: pid))
        }
        thread.name = "orbit.reaper.\(pid)"
        thread.start()
    }

    func exitStatus() async -> ExitStatus {
        await withCheckedContinuation { continuation in
            let finished: ExitStatus? = state.withLock { state in
                switch state {
                case .finished(let status):
                    return status
                case .running(let waiters):
                    state = .running(waiters + [continuation])
                    return nil
                }
            }
            if let finished {
                continuation.resume(returning: finished)
            }
        }
    }

    private func finish(with status: ExitStatus) {
        let waiters = state.withLock { state in
            guard case .running(let waiters) = state else { return [CheckedContinuation<ExitStatus, Never>]() }
            state = .finished(status)
            return waiters
        }
        waiters.forEach { $0.resume(returning: status) }
    }

    private static func wait(for pid: Int32) -> ExitStatus {
        var status: Int32 = 0
        while waitpid(pid, &status, 0) == -1 {
            guard errno == EINTR else { return .exited(-1) }
        }
        return ExitStatus(waitStatus: status)
    }
}
