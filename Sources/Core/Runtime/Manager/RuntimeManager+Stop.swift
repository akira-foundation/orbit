extension RuntimeManager {
    public func stop(_ projectID: String) async throws {
        attempts[projectID] = 0
        guard let session = sessions[projectID] else {
            await persist(.stopped, projectID: projectID)
            return
        }
        session.requestStop()
        var polls = Int((Self.stopGrace + .seconds(1)) / Self.stopPoll)
        while polls > 0 && session.status != .stopped && session.status != .error {
            polls -= 1
            do {
                try await clock.sleep(for: Self.stopPoll)
            } catch {
                break
            }
        }
        if let error = session.shutdownError {
            await persist(.error, session)
            await services.projectStopped(id: projectID)
            throw error
        }
        if session.status == .stopped {
            await persist(.stopped, session)
        }
        await services.projectStopped(id: projectID)
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
        for session in live where session.status.isLive {
            guard let supervisor = session.supervisor else { continue }
            try? inspector.signal(supervisor, .terminate)
        }
    }

    public func close() async {
        await stopAll()
        for task in Array(restarts.values) + Array(watchdogs.values) {
            task.cancel()
        }
        restarts.removeAll()
        watchdogs.removeAll()
    }
}
