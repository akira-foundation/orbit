import Darwin
import Foundation
import Testing

@testable import Core

@Suite(.serialized)
struct DarwinProcessInspectorTests {
    private let missingPID: Int32 = 1 << 30

    @Test func snapshotChangedBeforeEnrichmentIsAMismatchAndReadsNoFields() {
        let reader = ScriptedKernelReader(snapshots: [start("100:000001"), start("200:000001")])
        #expect(throws: RuntimeError.identityMismatch("changed before enrichment")) {
            try DarwinProcessInspector(reader: reader).identity(of: 4100)
        }
        #expect(reader.argumentReads == 0)
    }

    @Test func snapshotChangedDuringEnrichmentIsAMismatch() {
        let reader = ScriptedKernelReader(
            snapshots: [start("100:000001"), start("100:000001"), start("200:000001")]
        )
        #expect(throws: RuntimeError.identityMismatch("changed during enrichment")) {
            try DarwinProcessInspector(reader: reader).identity(of: 4100)
        }
        #expect(reader.argumentReads == 1)
    }

    @Test func reparentingDuringEnrichmentKeepsTheIdentity() throws {
        let reparented = KernelProcessSnapshot(
            pid: 4100, pgid: 4100, ppid: 1, startID: "100:000001", command: "fixture")
        let original = KernelProcessSnapshot(
            pid: 4100, pgid: 4100, ppid: 4000, startID: "100:000001", command: "fixture"
        )
        let reader = ScriptedKernelReader(snapshots: [original, original, reparented])
        let identity = try DarwinProcessInspector(reader: reader).identity(of: 4100)
        #expect(
            identity
                == ProcessIdentity(
                    pid: 4100, pgid: 4100, startID: "100:000001", executable: "fixture", arguments: "fixture 4100"
                )
        )
    }

    @Test func descendantsFollowParentsTransitivelyAndExcludeTheRoot() throws {
        let reader = ScriptedKernelReader(table: [
            member(13, parent: 15), member(10, parent: 1), member(12, parent: 10), member(15, parent: 12),
            member(11, parent: 10), member(20, parent: 1), member(0, parent: 10),
        ])
        let descendants = try DarwinProcessInspector(reader: reader).descendants(of: 10)
        #expect(descendants.map(\.pid) == [11, 12, 13, 15])
    }

    @Test func groupMembersAreUniqueAndSortedByPID() throws {
        let reader = ScriptedKernelReader(table: [
            member(12, parent: 10), member(10, parent: 1), member(12, parent: 10),
        ])
        let members = try DarwinProcessInspector(reader: reader).groupMembers(10)
        #expect(members.map(\.pid) == [10, 12])
    }

    @Test func groupMembersSkipsAMemberThatVanishedAfterListing() throws {
        let reader = ScriptedKernelReader(table: [member(10, parent: 1), member(12, parent: 10)], vanished: [12])
        #expect(try DarwinProcessInspector(reader: reader).groupMembers(10).map(\.pid) == [10])
    }

    @Test func descendantsSkipAMemberThatVanishedAfterListing() throws {
        let reader = ScriptedKernelReader(
            table: [member(10, parent: 1), member(11, parent: 10), member(12, parent: 10)], vanished: [11]
        )
        #expect(try DarwinProcessInspector(reader: reader).descendants(of: 10).map(\.pid) == [12])
    }

    @Test func listingsStillAbortOnOtherErrors() {
        let reader = ScriptedKernelReader(table: [member(10, parent: 1)], argumentFailure: "unreadable")
        #expect(throws: RuntimeError.message("unreadable")) {
            try DarwinProcessInspector(reader: reader).groupMembers(10)
        }
    }

    @Test func listingRereadsAMemberThatChangesTwiceAndThenSettles() throws {
        let reader = ScriptedKernelReader(
            snapshots: [moved(2), moved(3), moved(4), moved(5), moved(5), moved(5)],
            table: [moved(1)]
        )
        let members = try DarwinProcessInspector(reader: reader).groupMembers(10)
        #expect(members.map(\.startID) == ["5:000001"])
    }

    @Test func listingSkipsAMemberThatNeverSettlesAndKeepsTheOthers() throws {
        let reader = ScriptedKernelReader(table: [member(10, parent: 1), member(12, parent: 10)], unsettled: [12])
        let members = try DarwinProcessInspector(reader: reader).groupMembers(10)
        #expect(members.map(\.pid) == [10])
    }

    @Test func identityOfAProcessThatNeverSettlesIsStillAMismatch() {
        let reader = ScriptedKernelReader(table: [member(12, parent: 10)], unsettled: [12])
        #expect(throws: RuntimeError.identityMismatch("changed before enrichment")) {
            try DarwinProcessInspector(reader: reader).identity(of: 12)
        }
    }

    @Test func execBetweenSnapshotsTakesTheSettledCommandAndArguments() throws {
        let reader = ScriptedKernelReader(
            snapshots: [image("bash"), image("bash"), image("sleep"), image("sleep"), image("sleep")],
            arguments: ["bash -c loop", "/bin/sleep 30"]
        )
        let identity = try DarwinProcessInspector(reader: reader).identity(of: 4100)
        #expect(identity.executable == "sleep")
        #expect(identity.arguments == "/bin/sleep 30")
        #expect(reader.argumentReads == 2)
    }

    @Test func repeatedExecDuringEnrichmentIsAMismatch() {
        let reader = ScriptedKernelReader(
            snapshots: [image("bash"), image("bash"), image("sh"), image("sh"), image("sleep")]
        )
        #expect(throws: RuntimeError.identityMismatch("changed during enrichment")) {
            try DarwinProcessInspector(reader: reader).identity(of: 4100)
        }
        #expect(reader.argumentReads == 2)
    }

    @Test func descendantsFailureNamesTheRoot() {
        let reader = ScriptedKernelReader(tableFailure: "Cannot allocate memory")
        #expect(throws: RuntimeError.message("inspect descendants of PID 10: Cannot allocate memory")) {
            try DarwinProcessInspector(reader: reader).descendants(of: 10)
        }
    }

    @Test func identityRejectsNonPositivePID() {
        #expect(throws: RuntimeError.message("inspect process: invalid PID 0")) {
            try DarwinProcessInspector().identity(of: 0)
        }
    }

    @Test func groupMembersRejectsNonPositivePGID() {
        #expect(throws: RuntimeError.message("inspect process group: invalid PGID -1")) {
            try DarwinProcessInspector().groupMembers(-1)
        }
    }

    @Test func identityOfMissingPIDIsProcessGone() {
        #expect(throws: RuntimeError.processGone) {
            try DarwinProcessInspector().identity(of: missingPID)
        }
    }

    @Test func signalOfMissingPIDIsProcessGone() {
        let invented = ProcessIdentity(
            pid: missingPID, pgid: missingPID, startID: "1:000001", executable: "sleep", arguments: "/bin/sleep 30"
        )
        #expect(throws: RuntimeError.processGone) {
            try DarwinProcessInspector().signal(invented, .terminate)
        }
    }

    @Test func ownProcessIdentityIsConsistent() throws {
        let inspector = DarwinProcessInspector()
        let first = try inspector.identity(of: getpid())
        let second = try inspector.identity(of: getpid())
        #expect(first == second)
        #expect(first.pgid == getpgrp())
        #expect(!first.executable.isEmpty)
        #expect(!first.arguments.isEmpty)
    }

    @Test func liveSleepIsInItsGroupAndAmongDescendants() throws {
        let child = try SpawnedTestProcess.sleep()
        defer { child.terminate() }
        let inspector = DarwinProcessInspector()
        let identity = try inspector.identity(of: child.pid)
        #expect(identity.pgid == child.pid)
        #expect(identity.executable == "sleep")
        #expect(identity.arguments == "/bin/sleep 30")
        #expect(try inspector.groupMembers(child.pid) == [identity])
        #expect(try inspector.descendants(of: getpid()).contains(identity))
    }

    @Test func setuidChildHasEmptyArgumentsAndAStableIdentity() throws {
        let child = try SpawnedTestProcess.spawn("/usr/bin/top", ["-l", "0", "-s", "1"])
        defer { child.terminate() }
        let inspector = DarwinProcessInspector()
        let identity = try inspector.identity(of: child.pid)
        #expect(identity.arguments == "")
        #expect(identity.executable == "top")
        #expect(try inspector.identity(of: child.pid) == identity)
        #expect(try inspector.descendants(of: getpid()).contains(identity))
    }

    @Test func zombieIsProcessGoneAndLeavesTheListings() throws {
        let child = try SpawnedTestProcess.sleep()
        defer { child.terminate() }
        let inspector = DarwinProcessInspector()
        kill(child.pid, SIGKILL)
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: .seconds(2))
        var gone = false
        while !gone && clock.now < deadline {
            do {
                _ = try inspector.identity(of: child.pid)
                usleep(1000)
            } catch RuntimeError.processGone {
                gone = true
            }
        }
        #expect(gone)
        #expect(try inspector.groupMembers(child.pid) == [])
        #expect(try !inspector.descendants(of: getpid()).contains { $0.pid == child.pid })
    }

    @Test func signalDeliversToTheVerifiedProcess() throws {
        let child = try SpawnedTestProcess.sleep()
        defer { child.terminate() }
        let inspector = DarwinProcessInspector()
        try inspector.signal(inspector.identity(of: child.pid), .kill)
        var status: Int32 = 0
        #expect(waitpid(child.pid, &status, 0) == child.pid)
        #expect(status & 0x7f == SIGKILL)
    }

    @Test func signalRefusesAChangedIdentity() throws {
        let child = try SpawnedTestProcess.sleep()
        defer { child.terminate() }
        let inspector = DarwinProcessInspector()
        let actual = try inspector.identity(of: child.pid)
        let stale = ProcessIdentity(
            pid: actual.pid, pgid: actual.pgid, startID: "1:000001", executable: actual.executable,
            arguments: actual.arguments
        )
        #expect(throws: RuntimeError.identityMismatch("PID \(child.pid)")) {
            try inspector.signal(stale, .kill)
        }
        #expect(kill(child.pid, 0) == 0)
    }

    @Test(arguments: [Endpoint(host: "::1", port: 80), Endpoint(host: "127.0.0.1", port: 0)])
    func listenerOwnersRejectsNonIPv4(endpoint: Endpoint) {
        let expected = RuntimeError.message(
            "inspect listener: invalid IPv4 endpoint \(endpoint.host):\(endpoint.port)"
        )
        #expect(throws: expected) {
            try DarwinProcessInspector().listenerOwners(endpoint)
        }
    }

    @Test func listenerOwnersFindsTheListeningProcess() throws {
        let listener = try TestListener.open()
        defer { listener.close() }
        let inspector = DarwinProcessInspector()
        let owners = try inspector.listenerOwners(Endpoint(host: "127.0.0.1", port: listener.port))
        #expect(owners == [try inspector.identity(of: getpid())])
    }

    @Test func listenerOwnersAcceptsAnIPv4MappedAddress() throws {
        let listener = try TestListener.open()
        defer { listener.close() }
        let inspector = DarwinProcessInspector()
        let owners = try inspector.listenerOwners(Endpoint(host: "::ffff:127.0.0.1", port: listener.port))
        #expect(owners == [try inspector.identity(of: getpid())])
    }

    @Test func listenerOwnersSkipAnOwnerThatVanished() throws {
        let directory = try TemporaryDirectory()
        try directory.write("#!/bin/sh\nprintf 'p10\\np12\\n'\n", to: "lsof")
        let path = directory.path + "/lsof"
        #expect(chmod(path, 0o755) == 0)
        let inspector = DarwinProcessInspector(
            reader: ScriptedKernelReader(table: [member(10, parent: 1)]),
            lsof: LsofRunner(executable: URL(fileURLWithPath: path))
        )
        #expect(try inspector.listenerOwners(Endpoint(host: "127.0.0.1", port: 80)).map(\.pid) == [10])
    }

    @Test func listenerOwnersOfAClosedPortIsEmpty() throws {
        let listener = try TestListener.open()
        listener.close()
        let owners = try DarwinProcessInspector().listenerOwners(Endpoint(host: "127.0.0.1", port: listener.port))
        #expect(owners == [])
    }

    private func start(_ startID: String) -> KernelProcessSnapshot {
        KernelProcessSnapshot(pid: 4100, pgid: 4100, ppid: 1, startID: startID, command: "fixture")
    }

    private func moved(_ generation: Int) -> KernelProcessSnapshot {
        KernelProcessSnapshot(pid: 12, pgid: 10, ppid: 10, startID: "\(generation):000001", command: "fixture")
    }

    private func image(_ command: String) -> KernelProcessSnapshot {
        KernelProcessSnapshot(pid: 4100, pgid: 4100, ppid: 1, startID: "100:000001", command: command)
    }

    private func member(_ pid: Int32, parent: Int32) -> KernelProcessSnapshot {
        KernelProcessSnapshot(pid: pid, pgid: 10, ppid: parent, startID: "\(pid):000001", command: "fixture")
    }
}
