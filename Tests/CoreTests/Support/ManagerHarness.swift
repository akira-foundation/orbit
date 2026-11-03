import Foundation

@testable import Core

struct ManagerHarness {
    static let projectID = "p1"
    static let endpoint = Endpoint(host: "127.0.0.1", port: 41234)
    static let loginPath = "/usr/bin:/bin"

    let store: Store
    let directory: TemporaryDirectory
    let launcher: FakeLauncher
    let inspector: FakeProcessInspector
    let services: RecordingServices
    let sink = RecordingEventSink()
    let clock: ManualClock
    let events: EventLog
    let manager: RuntimeManager

    init(
        devCommand: String? = "/bin/echo ready", projects: [String] = [projectID],
        initialStatus: ProjectStatus = .stopped,
        base: [String] = [],
        serviceEnvironment: [String: String] = [:], holdServices: Bool = false,
        servicesHonourCancellation: Bool = false, stopIgnoredAt: Set<Int> = [],
        inspector: FakeProcessInspector = .supervised(), instantStopPolls: Bool = false
    ) async throws {
        clock = ManualClock(instantSleeps: instantStopPolls ? [RuntimeManager.stopPoll] : [])
        store = try Store.inMemory()
        directory = try TemporaryDirectory()
        launcher = FakeLauncher(stopIgnoredAt: stopIgnoredAt)
        self.inspector = inspector
        services = RecordingServices(
            environment: serviceEnvironment, holdStarts: holdServices, honoursCancellation: servicesHonourCancellation)
        events = EventLog(store: store, clock: clock)
        manager = RuntimeManager(
            store: store, launcher: launcher, inspector: inspector, endpoints: FixedEndpoint(endpoint: Self.endpoint),
            services: services,
            loginPath: LoginPathResolver(
                runner: StubShell(output: LoginPathResolver.marker + Self.loginPath), shell: "/bin/zsh", home: nil),
            events: events, sink: sink, clock: clock, baseEnvironment: EnvironmentList(base))
        for id in projects {
            try await store.upsert(
                Project(
                    id: id, name: id, path: try directory.makeDirectory(id), slug: id, localDomain: "\(id).orbit.test",
                    devCommand: devCommand, status: initialStatus))
        }
    }

    func status(_ id: String = projectID) async -> ProjectStatus {
        await manager.status(id).status
    }

    func persisted(_ id: String = projectID) async throws -> ProjectStatus? {
        try await store.project(id: id)?.status
    }

    func reaches(_ status: ProjectStatus, _ id: String = projectID) async -> Bool {
        await Poll.until { await self.status(id) == status }
    }

    func persists(_ status: ProjectStatus, _ id: String = projectID) async -> Bool {
        await Poll.until { try await self.persisted(id) == status }
    }

    func emitted(_ id: String = projectID) -> [RuntimeEventName] {
        sink.events.compactMap { event in
            guard case .status(let name, let snapshot) = event, snapshot.projectID == id else { return nil }
            return name
        }
    }

    func history(_ id: String = projectID) async throws -> [String] {
        try await manager.history(id, since: 0, limit: 0).map(\.text)
    }

    func exitAll() {
        launcher.handles.forEach { $0.exit() }
    }

    func startRunning(_ index: Int, _ id: String = projectID) async throws -> ScriptedHandle {
        let handle = try await launcher.handle(index)
        handle.write("  VITE ready in 300 ms")
        guard await reaches(.running, id) else {
            throw RuntimeError.message("session \(index) never reached running")
        }
        return handle
    }
}
