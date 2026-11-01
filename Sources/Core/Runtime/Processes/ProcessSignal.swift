import Darwin

public enum ProcessSignal: Sendable, Hashable {
    case terminate
    case kill
    case custom(Int32)

    public var rawValue: Int32 {
        switch self {
        case .terminate: SIGTERM
        case .kill: SIGKILL
        case .custom(let value): value
        }
    }
}
