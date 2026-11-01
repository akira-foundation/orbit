import Foundation

extension RuntimeManager {
    public func start(_ projectID: String) async throws {
        try await launch(projectID, automatic: false)
    }

    func launch(_ projectID: String, automatic: Bool) async throws {
        let session = try admit(projectID, automatic: automatic)
        let project: Project
        do {
            guard let found = try await store.project(id: projectID) else {
                throw StoreError.projectNotFound(projectID)
            }
            project = found
        } catch {
            throw await fail(session, "\(error)", reported: "runtime: load project: \(error)")
        }
        let command = CommandLineSplitter.split(project.devCommand ?? "")
        guard let devCommand = project.devCommand, let program = command.first else {
            throw await fail(session, "runtime: project has no dev command")
        }
        let serviceEnvironment = await startServices(project)
        let endpoint: Endpoint
        do {
            endpoint = try endpoints.reserve()
        } catch {
            throw await fail(session, "runtime: reserve endpoint: \(error)")
        }
        session.assign(endpoint)
        record(
            projectID: projectID, level: .info, source: .system,
            text: "starting runtime (cmd=\(Self.quoted(devCommand)) port=\(endpoint.port))")
        let environment = await childEnvironment(project, services: serviceEnvironment, endpoint: endpoint)
        let executable = ExecutableLookup.resolve(program, searchPath: PathList.value(in: environment))
        guard executable.contains("/") else {
            let error = "exec: \(Self.quoted(program)): executable file not found in $PATH"
            throw await fail(session, error, reported: "runtime: spawn: \(error)")
        }
        let handle: any SupervisorHandle
        do {
            handle = try await launcher.launch(
                SupervisorSpec(
                    sessionID: session.sessionID, token: OwnershipIdentifiers.token(), directory: project.path,
                    command: [executable] + command.dropFirst(), environment: environment, grace: Self.stopGrace))
        } catch {
            throw await fail(session, "\(error)", reported: "runtime: spawn: \(error)")
        }
        handles[session.sessionID] = handle
        session.setSupervisor(handle.identity)
        publish(.starting, session)
        await persist(.starting, session)
        supervise(session, handle)
    }

    private func admit(_ projectID: String, automatic: Bool) throws -> RuntimeSession {
        if let existing = sessions[projectID] {
            let status = existing.status
            if status.isLive {
                throw RuntimeError.message("runtime: already running")
            }
            let idle = clock.now.timeIntervalSince(existing.lastActivity)
            if !automatic && status == .error && idle < Double(Self.cooldown.components.seconds) {
                throw RuntimeError.message("runtime: cooling down after recent failure")
            }
            existing.requestStop()
        }
        if !automatic {
            attempts[projectID] = 0
        }
        let session = RuntimeSession(projectID: projectID, sessionID: OwnershipIdentifiers.sessionID(), clock: clock)
        sessions[projectID] = session
        return session
    }

    private func fail(_ session: RuntimeSession, _ message: String, reported: String? = nil) async -> RuntimeError {
        session.setError(message)
        record(projectID: session.projectID, level: .error, source: .system, text: message)
        if isCurrent(session) {
            publish(.error, session)
        }
        await persist(.error, session)
        return RuntimeError.message(reported ?? message)
    }

    private func startServices(_ project: Project) async -> [String: String] {
        let services = services
        let clock = clock
        let outcome = await withTaskGroup(of: Result<[String: String], any Error>?.self) { group in
            group.addTask {
                do {
                    return .success(
                        try await services.projectStarting(id: project.id, path: project.path, slug: project.slug))
                } catch {
                    return .failure(error)
                }
            }
            group.addTask {
                try? await clock.sleep(for: Self.servicesTimeout)
                return nil
            }
            let first = await group.next().flatMap { $0 }
            group.cancelAll()
            return first
        }
        switch outcome {
        case .success(let environment):
            return environment
        case .failure(let error):
            record(projectID: project.id, level: .warn, source: .system, text: "services start: \(error)")
        case nil:
            record(
                projectID: project.id, level: .warn, source: .system, text: "services start: context deadline exceeded")
        }
        return [:]
    }

    private func childEnvironment(
        _ project: Project, services: [String: String], endpoint: Endpoint
    ) async -> EnvironmentList {
        var environment = baseEnvironment
        environment.remove("CI")
        environment.set("FORCE_COLOR", "1")
        environment.addAbsent(services)
        environment.override(DotEnv.load(directory: project.path))
        environment.remove("HOST")
        environment.remove("PORT")
        environment.set("HOST", endpoint.host)
        environment.set("PORT", String(endpoint.port))
        return await loginPath.apply(to: environment)
    }

    private static func quoted(_ text: String) -> String {
        let escaped = text.replacingOccurrences(of: "\\", with: "\\\\").replacingOccurrences(of: "\"", with: "\\\"")
        return "\"\(escaped)\""
    }
}
