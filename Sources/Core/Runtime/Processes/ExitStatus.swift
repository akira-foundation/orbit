public enum ExitStatus: Sendable, Equatable {
    case exited(Int32)
    case signaled(Int32)

    init(waitStatus status: Int32) {
        let signal = status & 0x7f
        self = signal == 0 ? .exited((status >> 8) & 0xff) : .signaled(signal)
    }
}
