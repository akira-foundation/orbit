public enum ProcessGroupPlacement: Sendable, Equatable {
    case newLeader
    case newSession
    case join(Int32)
    case inherit
}
