import Darwin
import Foundation
import Synchronization

final class LaunchedSupervisor: SupervisorHandle {
    private static let pollInterval: Duration = .milliseconds(1)

    let identity: ProcessIdentity
    private let process: SpawnedProcess
    private let control: Int32
    private let inspector: any ProcessInspector
    private let stopped = Mutex<Result<Void, RuntimeError>?>(nil)

    init(process: SpawnedProcess, control: Int32, identity: ProcessIdentity, inspector: any ProcessInspector) {
        self.process = process
        self.control = control
        self.identity = identity
        self.inspector = inspector
    }

    var stdout: FileHandle? { process.stdout }
    var stderr: FileHandle? { process.stderr }

    func stop() throws {
        try stopped.withLock { result in
            if let result {
                return result
            }
            let closed: Result<Void, RuntimeError> =
                close(control) == 0
                ? .success(())
                : .failure(.message("runtime: close supervisor control: \(String(cString: strerror(errno)))"))
            result = closed
            return closed
        }.get()
    }

    func wait() async throws {
        if let failure = try await process.exitStatus().failure("supervisor") {
            throw failure
        }
    }

    func verifyEmpty(within limit: Duration) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        while try !inspector.groupMembers(identity.pgid).isEmpty {
            let notEmpty = RuntimeErrors(errors: [RuntimeError.groupNotEmpty, RuntimeError.cancelled])
            guard clock.now < deadline else { throw notEmpty }
            do {
                try await clock.sleep(for: Self.pollInterval)
            } catch {
                throw notEmpty
            }
        }
    }
}
