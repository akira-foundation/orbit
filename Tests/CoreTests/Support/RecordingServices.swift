import Synchronization

@testable import Core

final class RecordingServices: ServiceCoordinating {
    private struct State {
        var calls: [String] = []
        var holding: Bool
        var held: [CheckedContinuation<Void, Never>] = []
    }

    private let environment: [String: String]
    private let honoursCancellation: Bool
    private let state: Mutex<State>

    init(environment: [String: String] = [:], holdStarts: Bool = false, honoursCancellation: Bool = false) {
        self.environment = environment
        self.honoursCancellation = honoursCancellation
        state = Mutex(State(holding: holdStarts))
    }

    var calls: [String] { state.withLock { $0.calls } }

    var stopped: [String] {
        calls.filter { $0.hasPrefix("stopped ") }.map { String($0.dropFirst("stopped ".count)) }
    }

    var heldStarts: Int { state.withLock { $0.held.count } }

    func releaseStarts() {
        let held = state.withLock { state in
            state.holding = false
            defer { state.held.removeAll() }
            return state.held
        }
        held.forEach { $0.resume() }
    }

    func projectStarting(id: String, path: String, slug: String) async throws -> [String: String] {
        if honoursCancellation {
            await withTaskCancellationHandler {
                await passGate(id)
            } onCancel: {
                releaseStarts()
            }
            if Task.isCancelled {
                state.withLock { $0.calls.append("cancelled \(id)") }
                throw CancellationError()
            }
        } else {
            await passGate(id)
        }
        state.withLock { $0.calls.append("started \(id)") }
        return environment
    }

    private func passGate(_ id: String) async {
        await withCheckedContinuation { continuation in
            let holding = state.withLock { state in
                state.calls.append("starting \(id)")
                guard state.holding else { return false }
                state.held.append(continuation)
                return true
            }
            if !holding {
                continuation.resume()
            }
        }
    }

    func projectStopped(id: String) async {
        state.withLock { $0.calls.append("stopped \(id)") }
    }
}
