import Foundation
import Synchronization

@testable import Core

final class TestClock: RuntimeClock {
    private let current: Mutex<Date>

    init(_ start: Date = Date(timeIntervalSince1970: 1_800_000_000)) {
        current = Mutex(start)
    }

    var now: Date { current.withLock { $0 } }

    func advance(by interval: TimeInterval) {
        current.withLock { $0 = $0.addingTimeInterval(interval) }
    }

    func sleep(for duration: Duration) async throws {
        try await Task.sleep(for: .seconds(86_400))
    }
}
