struct ReapState {
    var status: ExitStatus?
    var waiters: [UInt64: CheckedContinuation<ExitStatus, any Error>] = [:]
    var cancelled: Set<UInt64> = []
    var nextWaiter: UInt64 = 0
}
