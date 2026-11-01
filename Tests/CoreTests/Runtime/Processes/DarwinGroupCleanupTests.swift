import Darwin
import Testing

@testable import Core

@Suite(.serialized)
struct DarwinGroupCleanupTests {
    private static let script = """
        i=0
        while [ $i -lt 20 ]; do
            /bin/sh -c "trap 'exit 0' TERM; while :; do /bin/sleep 0.00$((i % 9 + 1)); done" &
            i=$((i + 1))
        done
        /bin/sh -c "trap '' TERM; exec /bin/sleep 30" &
        exec /bin/sleep 30
        """

    @Test func cleanupKillsTheResistantMemberWhileOthersExit() async throws {
        for _ in 0..<5 {
            try await cleanOneGroup()
        }
    }

    private func cleanOneGroup() async throws {
        let leader = try SpawnedTestProcess.spawn("/bin/sh", ["-c", Self.script])
        defer { leader.terminate() }
        let inspector = DarwinProcessInspector()
        try await waitForMembers(of: leader.pid, count: 22, inspector: inspector)
        let supervisor = try inspector.identity(of: leader.pid)
        try await OwnedMemberCleanup.run(supervisor: supervisor, inspector: inspector, grace: .milliseconds(50))
        #expect(try inspector.groupMembers(leader.pid) == [supervisor])
    }

    private func waitForMembers(of pgid: Int32, count: Int, inspector: DarwinProcessInspector) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(5))
        while try inspector.groupMembers(pgid).count < count {
            guard clock.now < deadline else { throw RuntimeError.message("group \(pgid) never reached \(count)") }
            try await clock.sleep(for: .milliseconds(5))
        }
    }
}
