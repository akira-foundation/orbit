import Foundation
import Testing

@testable import Core

struct RuntimeSessionLogBudgetTests {
    @Test func logsAreLimitedByBytesAndKeepTheNewest() {
        let session = RuntimeSession(projectID: "p", sessionID: "s", clock: TestClock())
        let line = String(repeating: "x", count: StoredText.maximumBytes)
        let total = RuntimeSession.logByteBudget / StoredText.maximumBytes + 50
        for index in 0..<total {
            session.addLog(LogLine(timestamp: .now, stream: .stdout, text: "\(index) \(line)"))
        }
        let logs = session.logs()
        #expect(logs.reduce(0) { $0 + $1.text.utf8.count } <= RuntimeSession.logByteBudget)
        #expect(logs.last?.text.hasPrefix("\(total - 1) ") == true)
        #expect(logs.count < total)
        #expect(session.logTotal == total)
    }

    @Test func aLineLargerThanTheBudgetIsKeptAlone() {
        let session = RuntimeSession(projectID: "p", sessionID: "s", clock: TestClock())
        session.addLog(LogLine(timestamp: .now, stream: .stdout, text: "small"))
        session.addLog(
            LogLine(
                timestamp: .now, stream: .stdout, text: String(repeating: "x", count: RuntimeSession.logByteBudget + 1))
        )
        #expect(session.logs().count == 1)
        session.addLog(LogLine(timestamp: .now, stream: .stdout, text: "after"))
        #expect(session.logs().map(\.text) == ["after"])
    }

    @Test func evictedLogsLeaveOrderIntactAcrossManyWraps() {
        let session = RuntimeSession(projectID: "p", sessionID: "s", clock: TestClock())
        let total = RuntimeSession.logCapacity * 3 + 17
        for index in 0..<total {
            session.addLog(LogLine(timestamp: .now, stream: .stdout, text: "\(index)"))
        }
        let expected = ((total - RuntimeSession.logCapacity)..<total).map { "\($0)" }
        #expect(session.logs().map(\.text) == expected)
    }
}
