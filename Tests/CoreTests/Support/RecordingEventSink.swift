import Synchronization

@testable import Core

final class RecordingEventSink: RuntimeEventSink {
    private let recorded = Mutex<[RuntimeEvent]>([])

    var events: [RuntimeEvent] { recorded.withLock { $0 } }

    func emit(_ event: RuntimeEvent) {
        recorded.withLock { $0.append(event) }
    }
}
