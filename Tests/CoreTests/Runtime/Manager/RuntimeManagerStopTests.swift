import Foundation
import Synchronization
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

    @Test func closeCancelsPendingRestartsBeforeStoppingTheRest() async throws {
        let harness = try await ManagerHarness(
            projects: ["p1", "p2"], stopIgnoredAt: [1], inspector: .supervised(supervisor: 4101))
        try await harness.manager.start("p1")
        let crashing = try await harness.startRunning(0, "p1")
        try await harness.manager.start("p2")
        let stubborn = try await harness.startRunning(1, "p2")
        crashing.exit()
        #expect(await harness.clock.waitForPending(.seconds(1)))

        let close = Task { await harness.manager.close() }
        #expect(await harness.clock.waitForPending(RuntimeManager.stopAllTimeout))
        let restartsDuringClose = await harness.manager.restarts.count
        harness.clock.advance(by: 3)
        await close.value
        let refused: String?
        do {
            try await harness.manager.start("p1")
            refused = nil
        } catch {
            refused = String(describing: error)
        }

        #expect(restartsDuringClose == 0)
        #expect(harness.launcher.handles.count == 2)
        #expect(refused == "runtime: manager is closed")
        stubborn.exit()
    }

    private func failure(of start: Task<Void, any Error>) async -> String? {
        do {
            try await start.value
            return nil
        } catch {
            return String(describing: error)
        }
    }

    @Test func closeCancelsACooperativeServiceWithoutWaitingForItsTimeout() async throws {
        let harness = try await ManagerHarness(
            initialStatus: .error, holdServices: true, servicesHonourCancellation: true)
        let start = Task { try await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.services.heldStarts == 1 })

        let closed = Task { await harness.manager.close() }
        let closedInTime = await Poll.until {
            harness.services.calls.contains("stopped p1")
        }
        harness.services.releaseStarts()
        await closed.value
        let message = await failure(of: start)

        #expect(closedInTime)
        #expect(message == "runtime: manager is closed")
        #expect(harness.services.calls == ["starting p1", "cancelled p1", "stopped p1"])
        #expect(harness.launcher.specs.isEmpty)
        #expect(await harness.status() == .stopped)
        #expect(try await harness.persisted() == .stopped)
    }

    @Test func stopAllKeepsItsTimeoutAgainstAServiceThatIgnoresCancellation() async throws {
        let harness = try await ManagerHarness(initialStatus: .error, holdServices: true)
        let start = Task { try await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.services.heldStarts == 1 })

        let stopAll = Task { await harness.manager.stopAll() }
        #expect(await harness.clock.waitForPending(RuntimeManager.stopAllTimeout))
        harness.clock.advance(by: 3)
        let returned = Atomic<Bool>(false)
        let waiter = Task {
            await stopAll.value
            returned.store(true, ordering: .relaxed)
        }
        let returnedInTime = await Poll.until { returned.load(ordering: .relaxed) }
        let callsWhileHeld = harness.services.calls
        harness.services.releaseStarts()
        await waiter.value
        let message = await failure(of: start)

        #expect(returnedInTime)
        #expect(callsWhileHeld == ["starting p1"])
        #expect(message == "runtime: start cancelled")
        #expect(harness.launcher.specs.isEmpty)
        #expect(await Poll.until { harness.services.calls == ["starting p1", "started p1", "stopped p1"] })
        #expect(await harness.persists(.stopped))
    }

    @Test func stopGivesUpOnAServiceThatIgnoresCancellationWithinItsLimit() async throws {
        let harness = try await ManagerHarness(initialStatus: .error, holdServices: true, instantStopPolls: true)
        let start = Task { try await harness.manager.start(ManagerHarness.projectID) }
        #expect(await Poll.until { harness.services.heldStarts == 1 })

        let stop = Task { try await harness.manager.stop(ManagerHarness.projectID) }
        let returned = Atomic<Bool>(false)
        let waiter = Task {
            try await stop.value
            returned.store(true, ordering: .relaxed)
        }
        let returnedInTime = await Poll.until(within: .seconds(30)) { returned.load(ordering: .relaxed) }
        let callsWhileHeld = harness.services.calls
        harness.services.releaseStarts()
        try await waiter.value
        let message = await failure(of: start)

        #expect(returnedInTime)
        #expect(callsWhileHeld == ["starting p1"])
        #expect(message == "runtime: start cancelled")
        #expect(await Poll.until { harness.services.calls == ["starting p1", "started p1", "stopped p1"] })
        #expect(await harness.persists(.stopped))
    }

    @Test func closeWritesEveryRecordedEventBeforeItReturns() async throws {
        let directory = try TemporaryDirectory()
        let path = directory.path + "/orbit.sqlite"
        let store = try Store(path: path)
        let events = EventLog(store: store, clock: TestClock())
        let manager = RuntimeManager(
            store: store, launcher: FakeLauncher(), inspector: FakeProcessInspector.supervised(),
            endpoints: FixedEndpoint(endpoint: ManagerHarness.endpoint),
            loginPath: LoginPathResolver(runner: StubShell(output: nil), shell: "/bin/zsh", home: nil),
            events: events, sink: RecordingEventSink(), clock: TestClock(), baseEnvironment: EnvironmentList([]))
        for index in 0..<50 {
            await manager.record(projectID: nil, level: .error, source: .system, text: "line \(index)")
        }

        await manager.close()

        let reopened = try SQLiteDatabase(path: path)
        #expect(try reopened.query("SELECT COUNT(*) AS n FROM events;").first?.int("n") == 50)
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
