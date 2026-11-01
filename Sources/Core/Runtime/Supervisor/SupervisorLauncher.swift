import Darwin

public struct SupervisorLauncher: SupervisorLaunching {
    private static let controlDescriptor: Int32 = 3

    public let executable: String
    private let inspector: any ProcessInspector

    public init(executable: String, inspector: any ProcessInspector) {
        self.executable = executable
        self.inspector = inspector
    }

    public func launch(_ spec: SupervisorSpec) async throws -> any SupervisorHandle {
        let arguments = SupervisorArguments.make(
            SupervisorInvocation(
                sessionID: spec.sessionID, token: spec.token, controlDescriptor: Self.controlDescriptor,
                directory: spec.directory, command: spec.command))
        _ = try SupervisorArguments.parse(arguments)
        let control: (read: Int32, write: Int32)
        do {
            control = try PosixSpawner.makePipe()
        } catch {
            throw RuntimeError.message("runtime: create supervisor control pipe: \(error)")
        }
        let process: SpawnedProcess
        do {
            process = try PosixSpawner.spawn(
                SpawnRequest(
                    executable: executable, arguments: arguments,
                    environment: SupervisorGrace.environment(spec.environment ?? .process(), grace: spec.grace),
                    captureOutput: true, controlDescriptor: control.read))
        } catch {
            close(control.read)
            close(control.write)
            throw RuntimeError.message("runtime: start supervisor: \(error)")
        }
        close(control.read)
        do {
            let identity = try await startedIdentity(process.pid)
            return LaunchedSupervisor(
                process: process, control: control.write, identity: identity, inspector: inspector)
        } catch {
            close(control.write)
            _ = try? await process.exitStatus()
            throw error
        }
    }

    private func startedIdentity(_ pid: Int32) async throws -> ProcessIdentity {
        let identity: ProcessIdentity
        do {
            identity = try await inspector.leaderIdentity(of: pid)
        } catch let error where error.contains(.identityMismatch("")) || error.contains(.cancelled) {
            throw error
        } catch {
            throw RuntimeError.message("runtime: inspect started supervisor: \(error)")
        }
        guard identity.pid == pid, identity.pgid == pid else {
            throw RuntimeError.identityMismatch("supervisor PID \(pid) PGID \(identity.pgid)")
        }
        return identity
    }
}
