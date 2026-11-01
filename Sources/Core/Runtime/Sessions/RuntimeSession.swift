import Foundation
import Synchronization

public final class RuntimeSession: Sendable {
    public static let logCapacity = 2000

    public let projectID: String
    public let sessionID: String
    private let clock: any RuntimeClock
    private let state: Mutex<SessionState>

    public init(projectID: String, sessionID: String, clock: any RuntimeClock) {
        self.projectID = projectID
        self.sessionID = sessionID
        self.clock = clock
        state = Mutex(SessionState(startedAt: clock.now))
    }

    public var status: ProjectStatus { state.withLock { $0.status } }
    public var endpoint: Endpoint? { state.withLock { $0.endpoint } }
    public var supervisor: ProcessIdentity? { state.withLock { $0.supervisor } }
    public var shutdownError: (any Error)? { state.withLock { $0.shutdownError } }
    public var shutdownRequested: Bool { state.withLock { $0.shutdownRequested } }
    public var lastActivity: Date { state.withLock { $0.lastActivity } }
    public var wasRunning: Bool { state.withLock { $0.wasRunning } }
    public var logTotal: Int { state.withLock { $0.logTotal } }

    public func setStatus(_ status: ProjectStatus) {
        mutate { state, now in
            state.status = status
            if status == .running {
                state.readyAt = state.readyAt ?? now
                state.wasRunning = true
            }
        }
    }

    public func setError(_ message: String) {
        mutate { state, _ in
            state.errorMessage = message
            state.status = .error
        }
    }

    public func setShutdownError(_ error: any Error) {
        mutate { state, _ in
            state.shutdownError = error
            state.errorMessage = String(describing: error)
            state.status = .error
        }
    }

    public func markStopped() {
        mutate { state, _ in state.status = .stopped }
    }

    public func setSupervisor(_ identity: ProcessIdentity) {
        state.withLock { $0.supervisor = identity }
    }

    public func assign(_ endpoint: Endpoint) {
        mutate { state, _ in state.endpoint = endpoint }
    }

    public func addLog(_ line: LogLine) {
        mutate { state, _ in state.append(line, capacity: Self.logCapacity) }
    }

    public func logs() -> [LogLine] {
        state.withLock { $0.orderedLogs }
    }

    public func installStopCallback(_ callback: @escaping @Sendable () -> Void) {
        state.withLock { state in
            state.stopCallback = callback
            state.invokeStopCallback()
        }
    }

    public func requestShutdown() {
        state.withLock { state in
            state.shutdownRequested = true
            state.invokeStopCallback()
        }
    }

    public func requestStop() {
        let waiters = state.withLock { state in
            state.shutdownRequested = true
            state.invokeStopCallback()
            return state.releaseStop()
        }
        waiters.forEach { $0.resume() }
    }

    public func stopSignal() async {
        let id = state.withLock { state in
            defer { state.nextWaiterID += 1 }
            return state.nextWaiterID
        }
        await withTaskCancellationHandler {
            await withCheckedContinuation { continuation in
                let released = state.withLock { state in
                    guard !state.stopReleased, !Task.isCancelled else { return true }
                    state.stopWaiters[id] = continuation
                    return false
                }
                if released { continuation.resume() }
            }
        } onCancel: {
            state.withLock { $0.stopWaiters.removeValue(forKey: id) }?.resume()
        }
    }

    public func snapshot(conns: Int) -> SessionSnapshot {
        let now = clock.now
        return state.withLock { state in
            let live = state.status == .running || state.status == .starting
            return SessionSnapshot(
                projectID: projectID,
                status: state.status,
                pid: state.supervisor?.pid ?? 0,
                port: state.endpoint?.port ?? 0,
                startedAt: state.startedAt,
                readyAt: state.readyAt,
                uptimeMs: live ? Self.milliseconds(from: state.startedAt, to: now) : 0,
                startupMs: state.readyAt.map { Self.milliseconds(from: state.startedAt, to: $0) },
                conns: conns,
                lastActivity: state.lastActivity,
                error: state.errorMessage)
        }
    }

    private func mutate(_ change: (inout SessionState, Date) -> Void) {
        let now = clock.now
        state.withLock { state in
            change(&state, now)
            state.lastActivity = now
        }
    }

    private static func milliseconds(from start: Date, to end: Date) -> Int64 {
        Int64((end.timeIntervalSince(start) * 1000).rounded(.towardZero))
    }
}
