public enum RuntimeError: Error, Equatable, CustomStringConvertible {
    case processGone
    case identityMismatch(String)
    case groupNotEmpty
    case unownedEndpoint(String?)
    case unexpectedSupervisorExit
    case cancelled
    case message(String)

    public var description: String {
        switch self {
        case .processGone: "process already finished"
        case .identityMismatch(let detail): Self.detailed("runtime: process identity mismatch", detail)
        case .groupNotEmpty: "runtime: process group is not empty"
        case .unownedEndpoint(let detail):
            Self.detailed("runtime: endpoint listener is not owned by supervised group", detail)
        case .unexpectedSupervisorExit: "runtime: supervisor exited unexpectedly"
        case .cancelled: "cancelled"
        case .message(let text): text
        }
    }

    func matches(_ other: RuntimeError) -> Bool {
        switch (self, other) {
        case (.identityMismatch, .identityMismatch), (.unownedEndpoint, .unownedEndpoint): true
        default: self == other
        }
    }

    private static func detailed(_ base: String, _ detail: String?) -> String {
        guard let detail, !detail.isEmpty else { return base }
        return "\(base): \(detail)"
    }
}
