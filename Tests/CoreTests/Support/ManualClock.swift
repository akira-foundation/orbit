import Foundation
import Synchronization

@testable import Core

final class ManualClock: RuntimeClock {
    static let manualThreshold: Duration = .seconds(1)

    private struct Sleeper {
        let deadline: Date
        let duration: Duration
        let continuation: CheckedContinuation<Void, any Error>
    }

    private struct State {
        var now: Date
        var sleepers: [Int: Sleeper] = [:]
        var requested: [Duration] = []
        var nextID = 0
    }

    private let state: Mutex<State>
    private let instantSleeps: Set<Duration>

    init(_ start: Date = Date(timeIntervalSince1970: 1_800_000_000), instantSleeps: Set<Duration> = []) {
        state = Mutex(State(now: start))
        self.instantSleeps = instantSleeps
    }

    var now: Date { state.withLock { $0.now } }

    var requested: [Duration] { state.withLock { $0.requested } }

    func sleep(for duration: Duration) async throws {
        if instantSleeps.contains(duration) {
            try Task.checkCancellation()
            await Task.yield()
            return
        }
        guard duration >= Self.manualThreshold else {
            try await Task.sleep(for: duration)
            return
        }
        let id = state.withLock { state in
            defer { state.nextID += 1 }
            return state.nextID
        }
        try await withTaskCancellationHandler {
            try await withCheckedThrowingContinuation { continuation in
                let cancelled = state.withLock { state in
                    guard !Task.isCancelled else { return true }
                    state.requested.append(duration)
                    let seconds = Double(duration.components.seconds)
                    state.sleepers[id] = Sleeper(
                        deadline: state.now.addingTimeInterval(seconds), duration: duration, continuation: continuation)
                    return false
                }
                if cancelled {
                    continuation.resume(throwing: CancellationError())
                }
            }
        } onCancel: {
            state.withLock { $0.sleepers.removeValue(forKey: id) }?.continuation.resume(
                throwing: CancellationError())
        }
    }

    func advance(by interval: TimeInterval) {
        let due = state.withLock { state in
            state.now = state.now.addingTimeInterval(interval)
            let now = state.now
            let ready = state.sleepers.filter { $0.value.deadline <= now }
            for id in ready.keys {
                state.sleepers.removeValue(forKey: id)
            }
            return ready.values.map(\.continuation)
        }
        due.forEach { $0.resume() }
    }

    func waitForPending(_ duration: Duration, within limit: Duration = .seconds(20)) async -> Bool {
        await Poll.until(within: limit) {
            self.state.withLock { $0.sleepers.values.contains { $0.duration == duration } }
        }
    }
}
