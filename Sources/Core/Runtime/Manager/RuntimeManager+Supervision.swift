import Foundation

extension RuntimeManager {
    func supervise(_ session: RuntimeSession, _ handle: any SupervisorHandle) {
        session.installStopCallback { try? handle.stop() }
        for (stream, output) in [(LogStream.stdout, handle.stdout), (.stderr, handle.stderr)] {
            if let output {
                read(output, stream, session)
            }
        }
        watchdogs[session.sessionID] = Task { await self.watchReadiness(session) }
        Task { await self.awaitExit(session, handle) }
    }

    func received(_ text: String, _ stream: LogStream, _ session: RuntimeSession) async {
        log(LogLine(timestamp: clock.now, stream: stream, text: text), session)
        guard session.status == .starting, ReadinessDetector.looksReady(text) else { return }
        session.setStatus(.running)
        guard isCurrent(session) else { return }
        publish(.running, session)
        await persist(.running, session)
    }

    private nonisolated func read(_ output: FileHandle, _ stream: LogStream, _ session: RuntimeSession) {
        let chunks = AsyncStream<Data> { continuation in
            output.readabilityHandler = { readable in
                let data = readable.availableData
                guard !data.isEmpty else {
                    readable.readabilityHandler = nil
                    continuation.finish()
                    return
                }
                continuation.yield(data)
            }
        }
        Task.detached { [weak self] in
            var scanner = LineScanner()
            for await chunk in chunks {
                for line in scanner.feed(chunk) {
                    await self?.received(line, stream, session)
                }
            }
            for line in scanner.finish() {
                await self?.received(line, stream, session)
            }
        }
    }

    private func awaitExit(_ session: RuntimeSession, _ handle: any SupervisorHandle) async {
        var waitError: (any Error)?
        do {
            try await handle.wait()
        } catch {
            waitError = error
        }
        var verifyError: (any Error)?
        do {
            try await handle.verifyEmpty(within: Self.stopGrace + .seconds(1))
        } catch {
            verifyError = error
        }
        watchdogs.removeValue(forKey: session.sessionID)?.cancel()
        let requested = session.shutdownRequested
        var failure = RuntimeErrors.join([waitError, verifyError])
        if !requested {
            failure = RuntimeErrors.join([RuntimeError.unexpectedSupervisorExit, failure])
        }
        if let failure {
            session.setShutdownError(failure)
        }
        await finalize(session, failure, requested: requested)
        handles.removeValue(forKey: session.sessionID)
    }

    private func finalize(_ session: RuntimeSession, _ failure: (any Error)?, requested: Bool) async {
        guard isCurrent(session) else { return }
        guard session.status == .error else {
            session.markStopped()
            publish(.stopped, session)
            await persist(.stopped, session)
            return
        }
        publish(.error, session)
        if !requested {
            let message = failure.map { "process exited: \($0)" } ?? "process exited before becoming ready"
            log(LogLine(timestamp: clock.now, stream: .system, text: message), session)
            if session.wasRunning {
                scheduleRestart(session)
            } else {
                publish(.crash, session)
            }
        }
        await persist(.error, session)
    }

    private func scheduleRestart(_ session: RuntimeSession) {
        let projectID = session.projectID
        let attempt = (attempts[projectID] ?? 0) + 1
        attempts[projectID] = attempt
        guard let delay = RestartPolicy.delay(forAttempt: attempt) else {
            record(
                projectID: projectID, level: .error, source: .system,
                text: "self-heal giving up after \(RestartPolicy.maximumAttempts) attempts")
            publish(.crash, session)
            return
        }
        record(
            projectID: projectID, level: .warn, source: .system,
            text: "self-heal restart attempt \(attempt) scheduled in \(Self.text(delay))")
        restarts[projectID]?.cancel()
        restarts[projectID] = Task {
            guard await Self.elapses(delay, on: clock, before: session), isCurrent(session) else { return }
            try? await launch(projectID, automatic: true)
        }
    }

    private func watchReadiness(_ session: RuntimeSession) async {
        guard await Self.elapses(Self.readinessTimeout, on: clock, before: session), session.status == .starting
        else { return }
        let message = "watchdog killed runtime: never became ready in \(Self.text(Self.readinessTimeout))"
        record(projectID: session.projectID, level: .error, source: .system, text: message)
        session.setError(message)
        session.requestShutdown()
    }

    private static func elapses(
        _ duration: Duration, on clock: any RuntimeClock, before session: RuntimeSession
    ) async -> Bool {
        await withTaskGroup(of: Bool.self) { group in
            group.addTask { (try? await clock.sleep(for: duration)) != nil }
            group.addTask {
                await session.stopSignal()
                return false
            }
            let elapsed = await group.next() ?? false
            group.cancelAll()
            return elapsed
        }
    }

    private static func text(_ duration: Duration) -> String {
        let seconds = duration.components.seconds
        return seconds >= 60 ? "\(seconds / 60)m\(seconds % 60)s" : "\(seconds)s"
    }
}
