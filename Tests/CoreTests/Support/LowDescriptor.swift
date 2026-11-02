import Darwin

@testable import Core

enum LowDescriptor {
    private static let claimTimeout: Duration = .seconds(20)

    static func with<Result>(
        _ descriptor: Int32, placedOn target: Int32, _ body: () throws -> Result
    ) async throws -> Result {
        try await claim(target, copying: descriptor)
        defer { close(target) }
        return try body()
    }

    private static func claim(_ target: Int32, copying descriptor: Int32) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: claimTimeout)
        while clock.now < deadline {
            let copy = fcntl(descriptor, F_DUPFD_CLOEXEC, target)
            guard copy >= 0 else { throw RuntimeError.message("dup of \(descriptor) failed") }
            if copy == target {
                return
            }
            close(copy)
            try await clock.sleep(for: .milliseconds(1))
        }
        throw RuntimeError.message("descriptor \(target) never became free")
    }
}
