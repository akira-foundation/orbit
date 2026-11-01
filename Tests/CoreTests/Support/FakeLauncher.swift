import Synchronization

@testable import Core

final class FakeLauncher: SupervisorLaunching {
    private struct State {
        var specs: [SupervisorSpec] = []
        var handles: [ScriptedHandle] = []
        var failure: String?
        var holding = false
        var held: [CheckedContinuation<Void, Never>] = []
    }

    private let stopIgnoredAt: Set<Int>
    private let state = Mutex(State())

    init(stopIgnoredAt: Set<Int> = []) {
        self.stopIgnoredAt = stopIgnoredAt
    }

    var specs: [SupervisorSpec] { state.withLock { $0.specs } }

    var handles: [ScriptedHandle] { state.withLock { $0.handles } }

    var failure: String? {
        get { state.withLock { $0.failure } }
        set { state.withLock { $0.failure = newValue } }
    }

    var heldLaunches: Int { state.withLock { $0.held.count } }

    func holdLaunches() {
        state.withLock { $0.holding = true }
    }

    func releaseLaunches() {
        let held = state.withLock { state in
            state.holding = false
            defer { state.held.removeAll() }
            return state.held
        }
        held.forEach { $0.resume() }
    }

    func launch(_ spec: SupervisorSpec) async throws -> any SupervisorHandle {
        await withCheckedContinuation { continuation in
            let holding = state.withLock { state in
                guard state.holding else { return false }
                state.held.append(continuation)
                return true
            }
            if !holding {
                continuation.resume()
            }
        }
        return try state.withLock { state in
            if let failure = state.failure {
                throw RuntimeError.message(failure)
            }
            state.specs.append(spec)
            let index = state.handles.count
            let handle = ScriptedHandle(pid: 4100 + Int32(index), exitsOnStop: !stopIgnoredAt.contains(index))
            state.handles.append(handle)
            return handle
        }
    }

    func handle(_ index: Int) async throws -> ScriptedHandle {
        guard await Poll.until({ self.handles.count > index }) else {
            throw RuntimeError.message("launch \(index) never happened")
        }
        return handles[index]
    }
}
