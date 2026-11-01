import Foundation

struct SessionState {
    var status: ProjectStatus = .starting
    var supervisor: ProcessIdentity?
    var endpoint: Endpoint?
    let startedAt: Date
    var readyAt: Date?
    var lastActivity: Date
    var errorMessage: String?
    var shutdownError: (any Error)?
    var wasRunning = false
    var logs: [LogLine] = []
    var logHead = 0
    var logTotal = 0
    var shutdownRequested = false
    var stopCallback: (@Sendable () -> Void)?
    var stopCallbackInvoked = false
    var stopReleased = false
    var stopWaiters: [Int: CheckedContinuation<Void, Never>] = [:]
    var nextWaiterID = 0

    init(startedAt: Date) {
        self.startedAt = startedAt
        lastActivity = startedAt
    }

    var orderedLogs: [LogLine] {
        Array(logs[logHead...] + logs[..<logHead])
    }

    mutating func append(_ line: LogLine, capacity: Int) {
        if logs.count < capacity {
            logs.append(line)
        } else {
            logs[logHead] = line
            logHead = (logHead + 1) % capacity
        }
        logTotal += 1
    }

    mutating func invokeStopCallback() {
        guard shutdownRequested, let callback = stopCallback, !stopCallbackInvoked else { return }
        stopCallbackInvoked = true
        stopCallback = nil
        callback()
    }

    mutating func releaseStop() -> [CheckedContinuation<Void, Never>] {
        guard !stopReleased else { return [] }
        stopReleased = true
        defer { stopWaiters.removeAll() }
        return Array(stopWaiters.values)
    }
}
