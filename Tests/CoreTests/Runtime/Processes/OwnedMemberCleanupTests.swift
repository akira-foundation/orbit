import Testing

@testable import Core

struct OwnedMemberCleanupTests {
    @Test func cleanupEscalatesAndKeepsSupervisorAlive() async throws {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101, ignoresTerm: true)])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(10))
        #expect(fake.signals.map(\.1) == [.terminate, .kill])
        #expect(fake.signals.map(\.0) == [4101, 4101])
        #expect(fake.isAlive(4100) && !fake.isAlive(4101))
    }

    @Test func cleanupRejectsRecycledSupervisorPID() async {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101)])
        let expected = fake.identity(4100)
        fake.setStartID(4100, "different-start")
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: expected, inspector: fake, grace: .milliseconds(1))
        }
        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(fake.signals.isEmpty)
        #expect(fake.isAlive(4101))
    }

    @Test func cleanupRejectsRecycledMemberBeforeSignal() async {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101, recycleInsideSignal: true)])
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        }
        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(fake.signals.isEmpty)
    }

    @Test func cleanupReturnsSignalError() async {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101, denySignalOnce: "signal denied")])
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        }
        #expect(error.map { "\($0)" } == "signal denied")
    }

    @Test func cleanupSignalsLaterMembersAfterAnEarlierError() async {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, denySignalOnce: "signal denied"),
            .init(pid: 4102, ignoresTerm: true),
            .init(pid: 4103, denySignalOnce: "signal rejected"),
        ])
        let error = await #expect(throws: RuntimeErrors.self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        }
        #expect(error?.contains(.message("signal denied")) == true)
        #expect(error?.contains(.message("signal rejected")) == true)
        #expect(error.map { "\($0)" } == "signal denied\nsignal rejected")
        #expect(fake.signals.map(\.1) == [.terminate, .kill, .kill, .kill])
        #expect(!fake.isAlive(4101) && !fake.isAlive(4102) && !fake.isAlive(4103))
    }

    @Test func cleanupEscalatesAfterCancellation() async {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101, ignoresTerm: true)])
        let supervisor = fake.identity(4100)
        let task = Task {
            withUnsafeCurrentTask { $0?.cancel() }
            try await OwnedMemberCleanup.run(supervisor: supervisor, inspector: fake, grace: .seconds(1))
        }
        let clock = ContinuousClock()
        let started = clock.now
        let result = await task.result
        #expect(throws: RuntimeError.cancelled) { try result.get() }
        #expect(clock.now - started < .milliseconds(900))
        #expect(fake.signals.map(\.1) == [.terminate, .kill])
        #expect(!fake.isAlive(4101))
    }

    @Test func cleanupContinuesAfterMemberExitsBeforeSignal() async throws {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, exitBeforeSignal: true), .init(pid: 4102),
        ])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        #expect(!fake.isAlive(4101) && !fake.isAlive(4102))
        #expect(fake.signals.map(\.0) == [4102])
    }

    @Test func cleanupIncludesEscapedDescendant() async throws {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101, pgid: 9100, ppid: 4100)])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        #expect(!fake.isAlive(4101))
    }

    @Test func cleanupRetainsDescendantThatChangesGroupAfterTERM() async {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true, changeGroupAfterTerm: 9100)
        ])
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        }
        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(fake.isAlive(4101))
        #expect(fake.signals.map(\.1) == [.terminate])
    }

    @Test func cleanupDropsGroupNotEmptyFromTheTermPhase() async {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true),
            .init(pid: 4102, denySignalOnce: "signal denied"),
        ])
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        }
        #expect(error?.contains(.groupNotEmpty) == false)
        #expect(error.map { "\($0)" } == "signal denied")
    }

    @Test func cleanupFinalWaitIgnoresCancellation() async {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true, survivesKillForReads: 5)
        ])
        let supervisor = fake.identity(4100)
        let task = Task {
            withUnsafeCurrentTask { $0?.cancel() }
            try await OwnedMemberCleanup.run(supervisor: supervisor, inspector: fake, grace: .seconds(1))
        }
        let result = await task.result
        #expect(throws: RuntimeError.cancelled) { try result.get() }
        #expect(!fake.isAlive(4101))
    }

    @Test func cleanupVerifiesIdentityBeforeEverySignal() async {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true, changeGroupAfterTerm: 9100)
        ])
        fake.signalVerifiesIdentity = false
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(1))
        }
        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(fake.signals.map(\.1) == [.terminate])
        #expect(fake.isAlive(4101))
    }
}
