import Darwin
import Foundation
import Synchronization

final class ProcessReaper: Sendable {
    private let state = Mutex(ReapState())

    init(pid: Int32) {
        let thread = Thread { [self] in
            finish(with: Self.wait(for: pid))
        }
        thread.name = "orbit.reaper.\(pid)"
        thread.start()
    }

    var waiterCount: Int {
        state.withLock { $0.waiters.count }
    }

    func reapedStatus() -> ExitStatus? {
        state.withLock { $0.status }
    }

    func exitStatus() async throws -> ExitStatus {
        let waiter = state.withLock { state in
            defer { state.nextWaiter += 1 }
            return state.nextWaiter
        }
        return try await withTaskCancellationHandler {
            try await withCheckedThrowingContinuation { continuation in
                register(continuation, as: waiter)
            }
        } onCancel: {
            cancel(waiter)
        }
    }

    private func register(_ continuation: CheckedContinuation<ExitStatus, any Error>, as waiter: UInt64) {
        let outcome: Result<ExitStatus, any Error>? = state.withLock { state in
            if let status = state.status {
                return .success(status)
            }
            if state.cancelled.remove(waiter) != nil {
                return .failure(RuntimeError.cancelled)
            }
            state.waiters[waiter] = continuation
            return nil
        }
        outcome.map { continuation.resume(with: $0) }
    }

    private func cancel(_ waiter: UInt64) {
        let continuation = state.withLock { state in
            guard let continuation = state.waiters.removeValue(forKey: waiter) else {
                if state.status == nil {
                    state.cancelled.insert(waiter)
                }
                return CheckedContinuation<ExitStatus, any Error>?.none
            }
            return continuation
        }
        continuation?.resume(throwing: RuntimeError.cancelled)
    }

    private func finish(with status: ExitStatus) {
        let waiters = state.withLock { state in
            state.status = status
            state.cancelled.removeAll()
            defer { state.waiters.removeAll() }
            return Array(state.waiters.values)
        }
        waiters.forEach { $0.resume(returning: status) }
    }

    private static func wait(for pid: Int32) -> ExitStatus {
        var status: Int32 = 0
        while waitpid(pid, &status, 0) == -1 {
            guard errno == EINTR else { return .unknown(errno) }
        }
        return ExitStatus(waitStatus: status)
    }
}
