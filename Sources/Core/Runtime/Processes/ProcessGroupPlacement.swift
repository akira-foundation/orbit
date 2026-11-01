public enum ProcessGroupPlacement: Sendable, Equatable {
    case newLeader
    case join(Int32)
    case inherit
}
