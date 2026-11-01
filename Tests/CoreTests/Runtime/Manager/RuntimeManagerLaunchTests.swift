import Foundation
import Testing

@testable import Core

struct RuntimeManagerLaunchTests {
    private func isHex(_ value: String, count: Int) -> Bool {
        value.count == count && value.allSatisfy { $0.isHexDigit && !$0.isUppercase }
    }

    private func failure(_ body: () async throws -> Void) async -> String? {
        do {
            try await body()
            return nil
        } catch {
            return String(describing: error)
        }
    }

    @Test func mainCommandUsesSupervisor() async throws {
        let harness = try await ManagerHarness(devCommand: "/bin/echo ready")

        try await harness.manager.start(ManagerHarness.projectID)

        let spec = try #require(harness.launcher.specs.first)
        #expect(isHex(spec.sessionID, count: 16))
        #expect(isHex(spec.token, count: 32))
        #expect(spec.directory == harness.directory.path + "/" + ManagerHarness.projectID)
        #expect(spec.command == ["/bin/echo", "ready"])
        #expect(spec.grace == RuntimeManager.stopGrace)
        #expect(try await harness.persisted() == .starting)
        #expect(harness.emitted() == [.starting])
    }

    @Test func childEnvironmentHasOnlyTheAssignedEndpoint() async throws {
        let harness = try await ManagerHarness(
            base: ["CI=1", "HOST=x", "A=base"], serviceEnvironment: ["HOST": "::1", "PORT": "8888", "B": "svc"])
        try harness.directory.write("PORT=1\nA=dot", to: "\(ManagerHarness.projectID)/.env")

        try await harness.manager.start(ManagerHarness.projectID)

        let environment = try #require(harness.launcher.specs.first?.environment)
        #expect(environment.value(of: "CI") == nil)
        #expect(environment.value(of: "FORCE_COLOR") == "1")
        #expect(environment.value(of: "A") == "dot")
        #expect(environment.value(of: "B") == "svc")
        #expect(environment.value(of: "HOST") == "127.0.0.1")
        #expect(environment.value(of: "PORT") == "41234")
        #expect(environment.value(of: "PATH") == ManagerHarness.loginPath)
        #expect(environment.entries.filter { $0.hasPrefix("HOST=") }.count == 1)
        #expect(environment.entries.filter { $0.hasPrefix("PORT=") }.count == 1)
    }

    @Test func bareCommandIsResolvedAgainstTheChildPath() async throws {
        let harness = try await ManagerHarness(devCommand: "sleep 600")

        try await harness.manager.start(ManagerHarness.projectID)

        #expect(harness.launcher.specs.first?.command == ["/bin/sleep", "600"])
    }

    @Test func missingExecutableFailsAsSpawnError() async throws {
        let harness = try await ManagerHarness(devCommand: "orbit-no-such-command --flag")

        let message = await failure { try await harness.manager.start(ManagerHarness.projectID) }

        #expect(message?.hasPrefix("runtime: spawn: ") == true)
        #expect(harness.launcher.specs.isEmpty)
        #expect(await harness.status() == .error)
    }

    @Test func launchFailureIsASpawnError() async throws {
        let harness = try await ManagerHarness()
        harness.launcher.failure = "process already finished"

        let message = await failure { try await harness.manager.start(ManagerHarness.projectID) }

        #expect(message == "runtime: spawn: process already finished")
        #expect(await harness.status() == .error)
        #expect(try await harness.persisted() == .error)
        #expect(harness.emitted() == [.error])
    }

    @Test func readyLineMovesStartingToRunning() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)

        let handle = try await harness.startRunning(0)

        #expect(await harness.persists(.running))
        #expect(harness.emitted() == [.starting, .running])
        #expect(await harness.manager.logs(ManagerHarness.projectID).map(\.text) == ["  VITE ready in 300 ms"])
        #expect(await harness.manager.port(ManagerHarness.projectID) == 41234)
        #expect(await harness.manager.isRunning(ManagerHarness.projectID))
        #expect(await harness.manager.status(ManagerHarness.projectID).pid == handle.identity.pid)
        handle.exit()
    }

    @Test func stderrLinesAreWarnEvents() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.launcher.handle(0)

        handle.write("boom", to: .stderr)

        #expect(await Poll.until { try await harness.history().contains("boom") })
        let rows = try harness.store.db.query("SELECT level, stream, source FROM events WHERE text = 'boom';")
        #expect(rows.map { $0.string("level") } == ["warn"])
        #expect(rows.map { $0.string("stream") } == ["stderr"])
        #expect(rows.map { $0.string("source") } == ["runtime"])
        #expect(await harness.manager.logs(ManagerHarness.projectID).map(\.stream) == [.stderr])
        handle.exit()
    }

    @Test func noDevCommandFailsAndRecordsSystemLog() async throws {
        let harness = try await ManagerHarness(devCommand: nil)

        let message = await failure { try await harness.manager.start(ManagerHarness.projectID) }

        #expect(message == "runtime: project has no dev command")
        #expect(try await harness.history().first == "runtime: project has no dev command")
        #expect(await harness.status() == .error)
    }

    @Test func unknownProjectFailsWithLoadError() async throws {
        let harness = try await ManagerHarness()

        let message = await failure { try await harness.manager.start("ghost") }

        #expect(message?.hasPrefix("runtime: load project: ") == true)
        #expect(await harness.status("ghost") == .error)
    }

    @Test func alreadyRunningIsRefused() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)

        let message = await failure { try await harness.manager.start(ManagerHarness.projectID) }

        #expect(message == "runtime: already running")
        #expect(harness.launcher.specs.count == 1)
    }

    @Test func concurrentStartsOfOneProjectLaunchOnce() async throws {
        let harness = try await ManagerHarness()
        harness.launcher.holdLaunches()
        let first = Task { try await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.launcher.heldLaunches == 1 })

        let message = await failure { try await harness.manager.start(ManagerHarness.projectID) }
        harness.launcher.releaseLaunches()
        try await first.value

        #expect(message == "runtime: already running")
        #expect(harness.launcher.specs.count == 1)
    }

    @Test func stopDuringLaunchStopsTheHandleOnceLaunched() async throws {
        let harness = try await ManagerHarness()
        harness.launcher.holdLaunches()
        let start = Task { try await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.launcher.heldLaunches == 1 })

        let stop = Task { try await harness.manager.stop(ManagerHarness.projectID) }
        try await Task.sleep(for: .milliseconds(100))
        harness.launcher.releaseLaunches()
        try await start.value
        try await stop.value

        #expect(try await harness.launcher.handle(0).stopCalls == 1)
        #expect(await harness.status() == .stopped)
        #expect(try await harness.persisted() == .stopped)
    }

    @Test func userStartWithinCooldownIsRefused() async throws {
        let harness = try await ManagerHarness()
        harness.launcher.failure = "boom"
        _ = await failure { try await harness.manager.start(ManagerHarness.projectID) }
        harness.launcher.failure = nil

        harness.clock.advance(by: 9)
        let refused = await failure { try await harness.manager.start(ManagerHarness.projectID) }
        harness.clock.advance(by: 2)
        let accepted = await failure { try await harness.manager.start(ManagerHarness.projectID) }

        #expect(refused == "runtime: cooling down after recent failure")
        #expect(accepted == nil)
        #expect(harness.launcher.specs.count == 1)
    }
}
