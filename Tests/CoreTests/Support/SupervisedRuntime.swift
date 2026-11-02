import Darwin
import Foundation

@testable import Core

struct SupervisedRuntime {
    static let holds = "exec /bin/sleep 600"
    static let runnerExits = "/bin/sleep 1; exit 0"
    private static let settleLimit: Duration = .seconds(20)

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

    static func launch(_ spec: SupervisorSpec) async throws -> any SupervisorHandle {
        try await SupervisorLauncher(executable: ProcessFixture.runtimeFixturePath, inspector: DarwinProcessInspector())
            .launch(spec)
    }

    static func start(_ mode: String, grace: Duration) async throws -> SupervisedRuntime {
        let handle = try await launch(spec(["/bin/sh", "-c", script(mode)], grace: grace))
        do {
            let line = try readLine(handle, prefix: "GRANDCHILD_PID=")
            guard let pid = Int32(line.dropFirst("GRANDCHILD_PID=".count)) else {
                throw RuntimeError.message("unreadable grandchild line \(line)")
            }
            let grandchild = await settledSleeper(pid, waitForIt: true)
            if mode == holds {
                try await waitForSleepingMembers(of: handle.identity)
            }
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

    static func settledSleeper(
        _ pid: Int32, waitForIt: Bool, arguments: String = "/bin/sleep 600"
    ) async -> ProcessIdentity? {
        let inspector = DarwinProcessInspector()
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: settleLimit)
        while clock.now < deadline {
            guard let identity = try? inspector.identity(of: pid) else { return nil }
            if identity.arguments == arguments {
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
        while !runsMember(arguments, in: pgid) {
            guard clock.now < deadline else { throw RuntimeError.message("group \(pgid) never ran \(arguments)") }
            try await clock.sleep(for: .milliseconds(2))
        }
    }

    private static func runsMember(_ arguments: String, in pgid: Int32) -> Bool {
        ((try? DarwinProcessInspector().groupMembers(pgid)) ?? []).contains { $0.arguments == arguments }
    }

    static func waitForSleepingMembers(of supervisor: ProcessIdentity) async throws {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: settleLimit)
        while clock.now < deadline {
            let members = ((try? DarwinProcessInspector().groupMembers(supervisor.pgid)) ?? []).filter {
                $0.pid != supervisor.pid
            }
            if members.count == 2 && members.allSatisfy({ $0.arguments == "/bin/sleep 600" }) {
                return
            }
            try await clock.sleep(for: .milliseconds(2))
        }
        throw RuntimeError.message("group \(supervisor.pgid) never settled")
    }

    static func isGone(_ pid: Int32, was identity: ProcessIdentity?, within limit: Duration) async -> Bool {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        while clock.now < deadline {
            do {
                let snapshot = try KernelProcessReader().snapshot(pid)
                if let identity, snapshot.startID != identity.startID {
                    return true
                }
            } catch {
                return error.contains(.processGone)
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
        let inspector = DarwinProcessInspector()
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: OwnedMemberCleanup.killWait)
        while clock.now < deadline {
            guard let listing = try? inspector.groupListing(pgid),
                !listing.identities.isEmpty || !listing.unsettled.isEmpty
            else { return }
            listing.identities.forEach { try? inspector.signal($0, .kill) }
            usleep(1000)
        }
    }

    static func killVerified(_ identity: ProcessIdentity?) {
        guard let identity, (try? DarwinProcessInspector().identity(of: identity.pid)) == identity else { return }
        Darwin.kill(identity.pid, SIGKILL)
    }

    static func waitBounded(_ handle: any SupervisorHandle, within limit: Duration) async -> Result<Void, any Error>? {
        await withTaskGroup(of: Result<Void, any Error>?.self) { group in
            group.addTask {
                do {
                    try await handle.wait()
                    return .success(())
                } catch {
                    return .failure(error)
                }
            }
            group.addTask {
                try? await Task.sleep(for: limit)
                return nil
            }
            let first = await group.next()
            group.cancelAll()
            return first.flatMap { $0 }
        }
    }

    func grandchildGone(within limit: Duration = .seconds(20)) async -> Bool {
        await Self.isGone(grandchildPID, was: grandchild, within: limit)
    }

    func cleanUp() {
        Self.killGroup(handle.identity.pgid)
        Self.killVerified(grandchild)
    }
}
