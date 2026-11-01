import Darwin
import Foundation

@testable import Core

struct SupervisedRuntime {
    static let holds = "exec /bin/sleep 600"
    static let runnerExits = "exit 0"
    private static let settleLimit: Duration = .seconds(5)

    let handle: any SupervisorHandle
    let grandchildPID: Int32
    let grandchild: ProcessIdentity?

    static func script(_ mode: String) -> String {
        "trap '' TERM; (trap '' TERM; exec /bin/sleep 600) & echo GRANDCHILD_PID=$!; \(mode)"
    }

    static func spec(
        _ command: [String], grace: Duration, environment: EnvironmentList? = EnvironmentList([])
    ) -> SupervisorSpec {
        SupervisorSpec(
            sessionID: OwnershipIdentifiers.sessionID(), token: OwnershipIdentifiers.token(), directory: "/",
            command: command, environment: environment, grace: grace)
    }

    static func launch(_ spec: SupervisorSpec) throws -> any SupervisorHandle {
        try SupervisorLauncher(executable: ProcessFixture.runtimeFixturePath, inspector: DarwinProcessInspector())
            .launch(spec)
    }

    static func start(_ mode: String, grace: Duration) async throws -> SupervisedRuntime {
        let handle = try launch(spec(["/bin/sh", "-c", script(mode)], grace: grace))
        do {
            let line = try readLine(handle, prefix: "GRANDCHILD_PID=")
            guard let pid = Int32(line.dropFirst("GRANDCHILD_PID=".count)) else {
                throw RuntimeError.message("unreadable grandchild line \(line)")
            }
            let grandchild = await settledSleeper(pid, waitForIt: mode == holds)
            return SupervisedRuntime(handle: handle, grandchildPID: pid, grandchild: grandchild)
        } catch {
            killGroup(handle.identity.pgid)
            throw error
        }
    }

    static func readLine(_ handle: any SupervisorHandle, prefix: String) throws -> String {
        guard let descriptor = handle.stdout?.fileDescriptor else { throw RuntimeError.message("no stdout") }
        var scanner = LineScanner()
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: settleLimit)
        var buffer = [UInt8](repeating: 0, count: 4096)
        while clock.now < deadline {
            var poller = pollfd(fd: descriptor, events: Int16(POLLIN), revents: 0)
            guard poll(&poller, 1, 10) > 0 else { continue }
            let count = Darwin.read(descriptor, &buffer, buffer.count)
            guard count > 0 else { break }
            if let line = scanner.feed(buffer[0..<count]).first(where: { $0.hasPrefix(prefix) }) {
                return line
            }
        }
        throw RuntimeError.message("supervisor output had no \(prefix) line")
    }

    static func settledSleeper(_ pid: Int32, waitForIt: Bool) async -> ProcessIdentity? {
        let inspector = DarwinProcessInspector()
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: settleLimit)
        while clock.now < deadline {
            guard let identity = try? inspector.identity(of: pid) else { return nil }
            if identity.arguments == "/bin/sleep 600" {
                return identity
            }
            guard waitForIt else { return nil }
            try? await clock.sleep(for: .milliseconds(2))
        }
        return nil
    }

    static func waitForMember(_ arguments: String, in pgid: Int32) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: settleLimit)
        while try !DarwinProcessInspector().groupMembers(pgid).contains(where: { $0.arguments == arguments }) {
            guard clock.now < deadline else { throw RuntimeError.message("group \(pgid) never ran \(arguments)") }
            try await clock.sleep(for: .milliseconds(2))
        }
    }

    static func isGone(_ pid: Int32, was identity: ProcessIdentity?, within limit: Duration) async -> Bool {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        while clock.now < deadline {
            if Darwin.kill(pid, 0) == -1 && errno == ESRCH {
                return true
            }
            if let identity, (try? DarwinProcessInspector().identity(of: pid)) != identity {
                return true
            }
            try? await clock.sleep(for: .milliseconds(5))
        }
        return false
    }

    static func groupIsEmpty(_ pgid: Int32, within limit: Duration) async -> Bool {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        while clock.now < deadline {
            if (try? KernelProcessReader().group(pgid))?.isEmpty == true {
                return true
            }
            try? await clock.sleep(for: .milliseconds(5))
        }
        return false
    }

    static func killGroup(_ pgid: Int32) {
        if (try? KernelProcessReader().group(pgid))?.isEmpty == false {
            Darwin.kill(-pgid, SIGKILL)
        }
    }

    static func killVerified(_ identity: ProcessIdentity?) {
        guard let identity, (try? DarwinProcessInspector().identity(of: identity.pid)) == identity else { return }
        Darwin.kill(identity.pid, SIGKILL)
    }

    func grandchildGone(within limit: Duration = .seconds(3)) async -> Bool {
        await Self.isGone(grandchildPID, was: grandchild, within: limit)
    }

    func cleanUp() {
        Self.killGroup(handle.identity.pgid)
        Self.killVerified(grandchild)
    }
}
