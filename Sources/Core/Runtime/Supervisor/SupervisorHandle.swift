import Foundation

public protocol SupervisorHandle: Sendable {
    var identity: ProcessIdentity { get }
    var stdout: FileHandle? { get }
    var stderr: FileHandle? { get }
    func stop() throws
    func wait() async throws
    func verifyEmpty(within limit: Duration) async throws
}
