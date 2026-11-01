enum ReapState {
    case running([CheckedContinuation<ExitStatus, Never>])
    case finished(ExitStatus)
}
