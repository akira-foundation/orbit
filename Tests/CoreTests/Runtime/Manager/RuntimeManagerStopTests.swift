import Foundation
import Testing

@testable import Core

struct RuntimeManagerStopTests {
    @Test func requestedStopPersistsStopped() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let handle = try await harness.startRunning(0)

        try await harness.manager.stop(ManagerHarness.projectID)

        #expect(handle.stopCalls == 1)
        #expect(await harness.status() == .stopped)
        #expect(try await harness.persisted() == .stopped)
        #expect(harness.emitted() == [.starting, .running, .stopped])
        #expect(harness.services.stopped == [ManagerHarness.projectID])
        #expect(await !harness.manager.isRunning(ManagerHarness.projectID))
    }

    @Test func stopOfUnknownProjectPersistsStopped() async throws {
        let harness = try await ManagerHarness()
        try await harness.store.update(projectID: ManagerHarness.projectID) { $0.status = .running }

        try await harness.manager.stop(ManagerHarness.projectID)

        #expect(try await harness.persisted() == .stopped)
        #expect(await harness.status() == .stopped)
    }

    @Test func restartStopsTheSessionAndStartsANewOne() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let first = try await harness.startRunning(0)

        try await harness.manager.restart(ManagerHarness.projectID)

        #expect(first.stopCalls == 1)
        #expect(harness.launcher.specs.count == 2)
        #expect(harness.launcher.specs[0].sessionID != harness.launcher.specs[1].sessionID)
        #expect(await harness.status() == .starting)
        try await harness.launcher.handle(1).exit()
    }

    @Test func stopAllStopsEverySessionWithinTheSharedTimeout() async throws {
        let harness = try await ManagerHarness(
            projects: ["p1", "p2"], stopIgnoredAt: [1], inspector: .supervised(supervisor: 4101))
        try await harness.manager.start("p1")
        _ = try await harness.startRunning(0, "p1")
        try await harness.manager.start("p2")
        let stubborn = try await harness.startRunning(1, "p2")

        let stopAll = Task { await harness.manager.stopAll() }
        #expect(await harness.clock.waitForPending(RuntimeManager.stopAllTimeout))
        #expect(await harness.reaches(.stopped, "p1"))
        harness.clock.advance(by: 3)
        await stopAll.value

        #expect(await harness.status("p2") == .running)
        #expect(stubborn.stopCalls == 1)
        #expect(harness.inspector.signals.map(\.0) == [4101])
        #expect(harness.inspector.signals.map(\.1) == [.terminate])
        stubborn.exit()
    }

    @Test func connectionCountsIgnoreEmptyIDsAndDropAtZero() async throws {
        let harness = try await ManagerHarness()
        try await harness.manager.start(ManagerHarness.projectID)
        let manager = harness.manager

        await manager.connectionOpened("")
        await manager.connectionClosed("")
        await manager.connectionOpened(ManagerHarness.projectID)
        await manager.connectionOpened(ManagerHarness.projectID)
        let opened = await manager.status(ManagerHarness.projectID).conns
        await manager.connectionClosed(ManagerHarness.projectID)
        let afterOne = await manager.status(ManagerHarness.projectID).conns
        await manager.connectionClosed(ManagerHarness.projectID)
        await manager.connectionClosed(ManagerHarness.projectID)
        let afterAll = await manager.status(ManagerHarness.projectID).conns

        #expect(opened == 2)
        #expect(afterOne == 1)
        #expect(afterAll == 0)
        #expect(await manager.status("").conns == 0)
        try await harness.launcher.handle(0).exit()
    }

    @Test(.timeLimit(.minutes(2)))
    func realSupervisorRunsAndStopsAShellCommand() async throws {
        let store = try Store.inMemory()
        let directory = try TemporaryDirectory()
        try await store.upsert(
            Project(
                id: "real", name: "real", path: directory.path, slug: "real", localDomain: "real.orbit.test",
                devCommand: "/bin/sh -c \"echo VITE ready in 1 ms; exec /bin/sleep 600\""))
        let inspector = DarwinProcessInspector()
        let manager = RuntimeManager(
            store: store,
            launcher: SupervisorLauncher(executable: ProcessFixture.runtimeFixturePath, inspector: inspector),
            inspector: inspector, endpoints: FixedEndpoint(endpoint: ManagerHarness.endpoint),
            loginPath: LoginPathResolver(
                runner: StubShell(output: LoginPathResolver.marker + ManagerHarness.loginPath), shell: "/bin/zsh",
                home: nil),
            events: EventLog(store: store, clock: SystemClock()), sink: RecordingEventSink(),
            baseEnvironment: EnvironmentList([]))

        try await manager.start("real")
        let pgid = await manager.status("real").pid
        defer { SupervisedRuntime.killGroup(pgid) }
        #expect(pgid > 0)
        #expect(await Poll.until(within: .seconds(30)) { await manager.status("real").status == .running })

        try await manager.stop("real")

        #expect(await manager.status("real").status == .stopped)
        #expect(try await store.project(id: "real")?.status == .stopped)
        #expect(await SupervisedRuntime.groupIsEmpty(pgid, within: .seconds(10)))
    }
}
