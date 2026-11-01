import Synchronization

public final class StatusWrites: Sendable {
    private struct Sequence {
        var issued: [String: UInt64] = [:]
        var applied: [String: UInt64] = [:]
    }

    private let sequence = Mutex(Sequence())

    public init() {}

    public func ticket(for projectID: String) -> UInt64 {
        sequence.withLock { sequence in
            let next = (sequence.issued[projectID] ?? 0) + 1
            sequence.issued[projectID] = next
            return next
        }
    }

    public func write(_ status: ProjectStatus, projectID: String, ticket: UInt64, to store: Store) async {
        _ = try? await store.update(projectID: projectID) { project in
            if self.accept(ticket, for: projectID) {
                project.status = status
            }
        }
    }

    private func accept(_ ticket: UInt64, for projectID: String) -> Bool {
        sequence.withLock { sequence in
            guard ticket > sequence.applied[projectID] ?? 0 else { return false }
            sequence.applied[projectID] = ticket
            return true
        }
    }
}
