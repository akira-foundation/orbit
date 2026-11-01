public enum OwnedMemberCleanup {
    public static let pollInterval: Duration = .milliseconds(1)
    public static let killWait: Duration = .seconds(1)

    public static func run(
        supervisor: ProcessIdentity, inspector: any ProcessInspector, grace: Duration
    ) async throws {
        var members = try KnownRuntimeMembers(supervisor: supervisor, inspector: inspector)
        let termError = signal(members.ordered, except: supervisor.pid, with: .terminate, inspector: inspector)
        let waitError = await waitOnlySupervisor(supervisor, &members, inspector: inspector, limit: grace)
        guard let waitError else {
            if let termError { throw termError }
            return
        }
        guard shouldEscalate(waitError) else {
            if let error = RuntimeErrors.join([termError, waitError]) { throw error }
            return
        }
        let killError = signal(members.ordered, except: supervisor.pid, with: .kill, inspector: inspector)
        let killed = members
        let finalWaitError = await Task.detached {
            var remaining = killed
            return await waitOnlySupervisor(supervisor, &remaining, inspector: inspector, limit: killWait)
        }.value
        let retainedWaitError: (any Error)? = waitError.contains(.groupNotEmpty) ? nil : waitError
        if let error = RuntimeErrors.join([termError, retainedWaitError, killError, finalWaitError]) {
            throw error
        }
    }

    private static func signal(
        _ members: [ProcessIdentity], except supervisorPID: Int32, with signal: ProcessSignal,
        inspector: any ProcessInspector
    ) -> (any Error)? {
        var errors: [(any Error)?] = []
        for member in members where member.pid != supervisorPID {
            do {
                try inspector.verifyIdentity(member)
                try inspector.signal(member, signal)
            } catch  where !error.contains(.processGone) {
                errors.append(error)
            } catch {
                continue
            }
        }
        return RuntimeErrors.join(errors)
    }

    private static func waitOnlySupervisor(
        _ supervisor: ProcessIdentity, _ members: inout KnownRuntimeMembers, inspector: any ProcessInspector,
        limit: Duration
    ) async -> (any Error)? {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        while true {
            do {
                try members.refresh(supervisor: supervisor, inspector: inspector)
            } catch {
                return error
            }
            if members.containsOnly(supervisor.pid) {
                return nil
            }
            if Task.isCancelled {
                return RuntimeError.cancelled
            }
            if clock.now >= deadline {
                return RuntimeError.groupNotEmpty
            }
            do {
                try await clock.sleep(for: pollInterval)
            } catch {
                return RuntimeError.cancelled
            }
        }
    }

    private static func shouldEscalate(_ error: any Error) -> Bool {
        error.contains(.groupNotEmpty) || error.contains(.identityMismatch("")) || error.contains(.cancelled)
    }
}
