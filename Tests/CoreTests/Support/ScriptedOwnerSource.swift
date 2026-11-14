import Darwin
import Synchronization

@testable import Core

final class ScriptedOwnerSource: Sendable {
    private let owners: Mutex<[uid_t]>

    init(owners: [uid_t]) {
        self.owners = Mutex(owners)
    }

    func read(_ pid: Int32) throws -> uid_t {
        owners.withLock { $0.count > 1 ? $0.removeFirst() : $0[0] }
    }
}
