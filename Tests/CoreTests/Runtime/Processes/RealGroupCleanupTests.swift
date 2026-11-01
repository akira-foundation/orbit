import Darwin
import Testing

@testable import Core

struct RealGroupCleanupTests {
    private static let script = "(trap '' TERM; exec /bin/sleep 600) & echo ready; read _; exit 0"

    @Test func realGroupCleanupLeavesOnlyTheLeader() async throws {
        let input = try PosixSpawner.makePipe()
        var writeEnd: Int32? = input.write
        defer {
            close(input.read)
            if let writeEnd { close(writeEnd) }
        }
        let leader = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh", arguments: ["-c", Self.script], environment: EnvironmentList([]),
                captureOutput: true, standardInput: input.read))
        let fixture = SpawnedTestProcess(adopting: leader)
        defer { fixture.terminate() }
        let inspector = DarwinProcessInspector()
        try fixture.waitForLine("ready")
        try await waitForResistantMember(leader: leader.pid, inspector: inspector)

        try await OwnedMemberCleanup.run(
            supervisor: inspector.identity(of: leader.pid), inspector: inspector, grace: .milliseconds(20))
        #expect(try inspector.groupMembers(leader.pid).map(\.pid) == [leader.pid])

        close(input.write)
        writeEnd = nil
        #expect(await leader.exitStatus() == .exited(0))
        #expect(try inspector.groupMembers(leader.pid).isEmpty)
    }

    private func waitForResistantMember(leader: Int32, inspector: DarwinProcessInspector) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(5))
        while try !inspector.groupMembers(leader).contains(where: { $0.arguments == "/bin/sleep 600" }) {
            guard clock.now < deadline else { throw RuntimeError.message("group \(leader) never settled") }
            try await clock.sleep(for: .milliseconds(5))
        }
    }
}
