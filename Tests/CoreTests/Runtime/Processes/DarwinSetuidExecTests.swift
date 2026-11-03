import Darwin
import Testing

@testable import Core

@Suite(.serialized)
struct DarwinSetuidExecTests {
    private static let script = """
        i=0
        while [ $i -lt 4 ]; do
            /bin/sh -c 'while kill -0 $PPID 2>/dev/null; do /bin/ps -p $$ >/dev/null; done' &
            i=$((i + 1))
        done
        exec /bin/sleep 30
        """

    @Test func groupListingSurvivesMembersExecutingSetuidBinaries() async throws {
        let leader = try SpawnedTestProcess.spawn("/bin/sh", ["-c", Self.script])
        defer { leader.terminate() }
        let inspector = DarwinProcessInspector()
        try await waitForLeaderExec(leader.pid, inspector: inspector)
        var failures: [String] = []
        var passes = 0
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(5))
        while passes < 500 && clock.now < deadline {
            do {
                _ = try inspector.groupListing(leader.pid)
            } catch {
                failures.append("\(error)")
            }
            passes += 1
        }
        #expect(passes > 0)
        #expect(failures == [])
    }

    private func waitForLeaderExec(_ leader: Int32, inspector: DarwinProcessInspector) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(10))
        while try SettlingIdentity.executable(of: leader, inspector: inspector) != "sleep" {
            guard clock.now < deadline else { throw RuntimeError.message("leader \(leader) never executed sleep") }
            try await clock.sleep(for: .milliseconds(5))
        }
    }
}
