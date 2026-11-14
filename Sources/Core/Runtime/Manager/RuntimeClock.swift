import Foundation

public protocol RuntimeClock: Sendable {
    var now: Date { get }
    func sleep(for duration: Duration) async throws
}
