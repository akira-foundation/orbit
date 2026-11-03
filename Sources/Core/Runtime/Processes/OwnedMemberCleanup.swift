public enum OwnedMemberCleanup {
    public static let pollInterval: Duration = .milliseconds(1)
    public static let killWait: Duration = .seconds(1)

    public static func run(
        supervisor: ProcessIdentity, inspector: any ProcessInspector, grace: Duration
    ) async throws {
        var members = try KnownRuntimeMembers(supervisor: supervisor, inspector: inspector)
        var terminated: Set<Int32> = []
        let termError = signal(
            members.ordered, except: supervisor.pid, with: .terminate, signalled: &terminated, inspector: inspector
        )
        let termWait = await drainUntilOnlySupervisor(
            supervisor, &members, phase: .terminate, signalled: &terminated, inspector: inspector, limit: grace
        )
        guard let endError = termWait.end, shouldEscalate(endError) else {
            if let error = RuntimeErrors.join([termError, termWait.late, termWait.end]) { throw error }
            return
        }
        var killed: Set<Int32> = []
        let killError = signal(
            members.ordered, except: supervisor.pid, with: .kill, signalled: &killed, inspector: inspector
        )
        let survivors = members
        let alreadyKilled = killed
        let killWaitResult = await Task.detached {
            var remaining = survivors
            var signalled = alreadyKilled
            return await drainUntilOnlySupervisor(
                supervisor, &remaining, phase: .kill, signalled: &signalled, inspector: inspector, limit: killWait
            )
        }.value
        let retainedEndErrors = ((endError as? RuntimeErrors)?.errors ?? [endError]).filter {
            !$0.contains(.groupNotEmpty)
        }
        let errors =
            [termError, termWait.late] + retainedEndErrors + [killError, killWaitResult.late, killWaitResult.end]
        if let error = RuntimeErrors.join(errors) {
            throw error
        }
    }

    private static func signal(
        _ members: [ProcessIdentity], except supervisorPID: Int32, with signal: ProcessSignal,
        signalled: inout Set<Int32>, inspector: any ProcessInspector
    ) -> (any Error)? {
        var errors: [(any Error)?] = []
        for member in members where member.pid != supervisorPID && !signalled.contains(member.pid) {
            let attempt = deliver(signal, to: member, inspector: inspector)
            if attempt.done {
                signalled.insert(member.pid)
            }
            errors.append(attempt.error)
        }
        return RuntimeErrors.join(errors)
    }

    private static func deliver(
        _ signal: ProcessSignal, to member: ProcessIdentity, inspector: any ProcessInspector
    ) -> (done: Bool, error: (any Error)?) {
        do {
            let current = try inspector.identity(of: member.pid)
            guard current.isSameProcess(as: member) else {
                return (true, RuntimeError.identityMismatch("PID \(member.pid)"))
            }
            try inspector.signal(current, signal)
            return (true, nil)
        } catch let error where error.contains(.processGone) {
            return (true, nil)
        } catch let error where error.contains(.identityMismatch("")) {
            let stillOurs = (try? inspector.identity(of: member.pid))?.isSameProcess(as: member) ?? true
            return stillOurs ? (false, nil) : (true, error)
        } catch {
            return (true, error)
        }
    }

    private static func drainUntilOnlySupervisor(
        _ supervisor: ProcessIdentity, _ members: inout KnownRuntimeMembers, phase: ProcessSignal,
        signalled: inout Set<Int32>, inspector: any ProcessInspector, limit: Duration
    ) async -> (late: (any Error)?, end: (any Error)?) {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        var lateErrors: [(any Error)?] = []
        while true {
            do {
                try members.refresh(supervisor: supervisor, inspector: inspector)
            } catch {
                return (RuntimeErrors.join(lateErrors), error)
            }
            lateErrors.append(
                signal(
                    members.ordered, except: supervisor.pid, with: phase, signalled: &signalled, inspector: inspector)
            )
            if members.containsOnly(supervisor.pid) {
                return (RuntimeErrors.join(lateErrors), nil)
            }
            if Task.isCancelled {
                return (RuntimeErrors.join(lateErrors), RuntimeError.cancelled)
            }
            if clock.now >= deadline {
                return (RuntimeErrors.join(lateErrors), deadlineError(members, phase: phase))
            }
            do {
                try await clock.sleep(for: pollInterval)
            } catch {
                return (RuntimeErrors.join(lateErrors), RuntimeError.cancelled)
            }
        }
    }

    private static func deadlineError(_ members: KnownRuntimeMembers, phase: ProcessSignal) -> RuntimeError {
        guard phase == .terminate || members.unsettled.isEmpty else {
            let pids = members.unsettled.sorted().map(String.init).joined(separator: ", ")
            return RuntimeError.identityMismatch("members did not settle: \(pids)")
        }
        return RuntimeError.groupNotEmpty
    }

    private static func shouldEscalate(_ error: any Error) -> Bool {
        error.contains(.groupNotEmpty) || error.contains(.identityMismatch("")) || error.contains(.cancelled)
    }
}
