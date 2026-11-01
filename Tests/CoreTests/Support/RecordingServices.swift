import Synchronization

@testable import Core

final class RecordingServices: ServiceCoordinating {
    private let environment: [String: String]
    private let recorded = Mutex<[String]>([])

    init(environment: [String: String] = [:]) {
        self.environment = environment
    }

    var stopped: [String] { recorded.withLock { $0 } }

    func projectStarting(id: String, path: String, slug: String) async throws -> [String: String] {
        environment
    }

    func projectStopped(id: String) async {
        recorded.withLock { $0.append(id) }
    }
}
