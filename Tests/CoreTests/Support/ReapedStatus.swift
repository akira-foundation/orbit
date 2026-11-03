@testable import Core

enum ReapedStatus {
    static func await(_ process: SpawnedProcess, within limit: Duration = .seconds(2)) async throws -> ExitStatus {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        while clock.now < deadline {
            if let status = process.reaper.reapedStatus() {
                return status
            }
            try await clock.sleep(for: .milliseconds(2))
        }
        throw RuntimeError.message("PID \(process.pid) was not reaped within \(limit)")
    }
}
