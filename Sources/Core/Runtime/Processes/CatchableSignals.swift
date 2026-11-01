import Darwin

enum CatchableSignals {
    static let all: sigset_t = (1..<NSIG)
        .filter { $0 != SIGKILL && $0 != SIGSTOP }
        .reduce(sigset_t(0)) { $0 | (1 << ($1 - 1)) }
}
