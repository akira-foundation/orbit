import Darwin
import Testing

@testable import Core

@Suite(.serialized)
struct DarwinGroupCleanupTests {
    private static let script = """
        i=0
        while [ $i -lt 20 ]; do
            /bin/sh -c "trap 'exit 0' TERM; while kill -0 \\$PPID 2>/dev/null; do /bin/sleep 0.00$((i % 9 + 1)); done" &
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

    @Test func memberThatExecsBetweenPassesIsKilledAndTheGroupEmpties() async throws {
        let script = """
            /bin/sh -c "trap '' TERM; /bin/sleep 0.5; exec /bin/sleep 30" &
            exec /bin/sleep 30
            """
        let leader = try SpawnedTestProcess.spawn("/bin/sh", ["-c", script])
        defer { leader.terminate() }
        let inspector = DarwinProcessInspector()
        try await SupervisedRuntime.waitForMember("/bin/sleep 0.5", in: leader.pid)
        let supervisor = try await inspector.leaderIdentity(of: leader.pid)
        try await OwnedMemberCleanup.run(supervisor: supervisor, inspector: inspector, grace: .seconds(1))
        #expect(try inspector.groupMembers(leader.pid) == [supervisor])
    }

    @Test func memberWithDestroyedArgumentsDoesNotAbortCleanup() async throws {
        let script = """
            '\(ProcessFixture.runtimeFixturePath)' --clobber-arguments &
            /bin/sleep 30 &
            exec /bin/sleep 30
            """
        let leader = try SpawnedTestProcess.spawn("/bin/sh", ["-c", script])
        defer { leader.terminate() }
        let inspector = DarwinProcessInspector()
        #expect(
            await Poll.until {
                try inspector.groupMembers(leader.pid).contains {
                    $0.executable.hasPrefix("orbit") && $0.arguments == ""
                }
            })
        try await SupervisedRuntime.waitForMember("/bin/sleep 30", in: leader.pid)
        let supervisor = try await inspector.leaderIdentity(of: leader.pid)
        try await OwnedMemberCleanup.run(supervisor: supervisor, inspector: inspector, grace: .seconds(1))
        #expect(try inspector.groupMembers(leader.pid) == [supervisor])
    }

    private func cleanOneGroup() async throws {
        let leader = try SpawnedTestProcess.spawn("/bin/sh", ["-c", Self.script])
        defer { leader.terminate() }
        let inspector = DarwinProcessInspector()
        try await waitUntilSettled(leader: leader.pid, inspector: inspector)
        let supervisor = try inspector.identity(of: leader.pid)
        try await OwnedMemberCleanup.run(supervisor: supervisor, inspector: inspector, grace: .milliseconds(50))
        #expect(try inspector.groupMembers(leader.pid) == [supervisor])
    }

    private func waitUntilSettled(leader: Int32, inspector: DarwinProcessInspector) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(10))
        while try !isSettled(leader: leader, inspector: inspector) {
            guard clock.now < deadline else { throw RuntimeError.message("group \(leader) never settled") }
            try await clock.sleep(for: .milliseconds(5))
        }
    }

    private func isSettled(leader: Int32, inspector: DarwinProcessInspector) throws -> Bool {
        guard try SettlingIdentity.executable(of: leader, inspector: inspector) == "sleep" else { return false }
        return try inspector.groupListing(leader).identities.contains {
            $0.pid != leader && $0.executable == "sleep" && $0.arguments == "/bin/sleep 30"
        }
    }
}
