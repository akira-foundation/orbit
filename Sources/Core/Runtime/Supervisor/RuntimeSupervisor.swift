import Darwin
import Dispatch
import Foundation

public enum RuntimeSupervisor {
    private static let parentPollInterval: Duration = .milliseconds(100)
    private static let stopSignals = [SIGTERM, SIGINT]

    private typealias Triggers = AsyncStream<SupervisorTrigger>.Continuation

    public static func runIfRequested(arguments: [String]) -> Int32? {
        guard arguments.first == SupervisorArguments.mode else { return nil }
        let outcome = SupervisorOutcome()
        Task.detached {
            do {
                try await run(SupervisorArguments.parse(arguments))
                outcome.finish(nil)
            } catch {
                outcome.finish(error)
            }
        }
        guard let error = outcome.wait() else { return 0 }
        FileHandle.standardError.write(Data("\(error)\n".utf8))
        return 1
    }

    private static func run(_ invocation: SupervisorInvocation) async throws {
        let parent = getppid()
        let (triggers, continuation) = AsyncStream.makeStream(of: SupervisorTrigger.self)
        signal(SIGPIPE, SIG_IGN)
        let sources = stopSignals.map { watchSignal($0, continuation) }
        defer { sources.forEach { $0.cancel() } }
        guard fcntl(invocation.controlDescriptor, F_SETFD, FD_CLOEXEC) != -1 else {
            throw RuntimeError.message("runtime: supervisor control descriptor unavailable")
        }
        let inspector = DarwinProcessInspector()
        let identity = try ownIdentity(inspector)
        let child: SpawnedProcess
        do {
            child = try PosixSpawner.spawn(
                SpawnRequest(
                    executable: invocation.command[0], arguments: Array(invocation.command.dropFirst()),
                    environment: .process(), directory: invocation.directory, group: .join(identity.pgid)))
        } catch {
            throw RuntimeError.message("runtime: start supervised command: \(error)")
        }
        watchControl(invocation.controlDescriptor, continuation)
        let watchers = [
            Task {
                guard let status = try? await child.exitStatus() else { return }
                continuation.yield(.childExited(status))
            },
            Task { await watchParent(parent, continuation) },
        ]
        defer { watchers.forEach { $0.cancel() } }
        var pending = triggers.makeAsyncIterator()
        let trigger = await pending.next() ?? .stopRequested
        try await shutdown(trigger, child: child, identity: identity, inspector: inspector)
    }

    private static func ownIdentity(_ inspector: some ProcessInspector) throws -> ProcessIdentity {
        let identity: ProcessIdentity
        do {
            identity = try inspector.identity(of: getpid())
        } catch {
            throw RuntimeError.message("runtime: inspect supervisor identity: \(error)")
        }
        guard identity.pgid == identity.pid else {
            throw RuntimeError.identityMismatch("supervisor is not group leader")
        }
        return identity
    }

    private static func shutdown(
        _ trigger: SupervisorTrigger, child: SpawnedProcess, identity: ProcessIdentity,
        inspector: some ProcessInspector
    ) async throws {
        var cleanupError: (any Error)?
        do {
            let grace = SupervisorGrace.parse(ProcessInfo.processInfo.environment[SupervisorGrace.variable])
            try await OwnedMemberCleanup.run(supervisor: identity, inspector: inspector, grace: grace)
        } catch {
            cleanupError = error
        }
        switch trigger {
        case .childExited(let status):
            if let error = RuntimeErrors.join([cleanupError, status.failure("runtime: supervised command")]) {
                throw error
            }
        case .stopRequested:
            _ = try await child.exitStatus()
            if let cleanupError {
                throw cleanupError
            }
        }
    }

    private static func watchSignal(_ number: Int32, _ continuation: Triggers) -> any DispatchSourceSignal {
        signal(number, SIG_IGN)
        let source = DispatchSource.makeSignalSource(signal: number, queue: .global())
        source.setEventHandler { continuation.yield(.stopRequested) }
        source.resume()
        return source
    }

    private static func watchControl(_ descriptor: Int32, _ continuation: Triggers) {
        let thread = Thread {
            var buffer = [UInt8](repeating: 0, count: 512)
            while true {
                let count = read(descriptor, &buffer, buffer.count)
                if count > 0 || (count == -1 && errno == EINTR) {
                    continue
                }
                break
            }
            continuation.yield(.stopRequested)
        }
        thread.name = "orbit.supervisor.control"
        thread.start()
    }

    private static func watchParent(_ parent: pid_t, _ continuation: Triggers) async {
        while !Task.isCancelled {
            if getppid() != parent {
                continuation.yield(.stopRequested)
                return
            }
            try? await Task.sleep(for: parentPollInterval)
        }
    }
}
