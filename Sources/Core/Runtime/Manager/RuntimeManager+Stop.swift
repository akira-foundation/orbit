extension RuntimeManager {
    public func stop(_ projectID: String) async throws {
        attempts[projectID] = 0
        guard let session = sessions[projectID] else {
            await persist(.stopped, projectID: projectID)
            return
        }
        session.requestStop()
        var polls = Int((Self.stopGrace + .seconds(1)) / Self.stopPoll)
        while polls > 0 && !isSettled(session) {
            polls -= 1
            do {
                try await clock.sleep(for: Self.stopPoll)
            } catch {
                break
            }
        }
        guard !launching.contains(session.sessionID) else {
            abandonedLaunches.insert(session.sessionID)
            return
        }
        if let error = session.shutdownError {
            await persist(.error, session)
            await services.projectStopped(id: projectID)
            throw error
        }
        await settleStop(session)
    }

    func settleStop(_ session: RuntimeSession) async {
        if session.status == .stopped {
            await persist(.stopped, session)
        }
        await services.projectStopped(id: session.projectID)
    }

    private func isSettled(_ session: RuntimeSession) -> Bool {
        !launching.contains(session.sessionID) && (session.status == .stopped || session.status == .error)
    }

    public func restart(_ projectID: String) async throws {
        try await stop(projectID)
        try await clock.sleep(for: Self.restartPause)
        try await start(projectID)
    }

    public func stopAll() async {
        let live = sessions.values.filter { $0.status.isLive }
        let projectIDs = live.map(\.projectID)
        let clock = clock
        await withTaskGroup(of: Void.self) { group in
            group.addTask {
                await withTaskGroup(of: Void.self) { stops in
                    for projectID in projectIDs {
                        stops.addTask { try? await self.stop(projectID) }
                    }
                }
            }
            group.addTask { try? await clock.sleep(for: Self.stopAllTimeout) }
            await group.next()
            group.cancelAll()
        }
        for session in supervising.values {
            guard let supervisor = session.supervisor else { continue }
            try? inspector.signal(supervisor, .terminate)
        }
    }

    public func close() async {
        closing = true
        restarts.values.forEach { $0.cancel() }
        restarts.removeAll()
        await stopAll()
        watchdogs.values.forEach { $0.cancel() }
        watchdogs.removeAll()
    }
}
