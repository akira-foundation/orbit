import Testing

@testable import Core

struct OwnedMemberCleanupExecTests {
    @Test func aMemberThatExecsBetweenPassesReceivesKillUnderItsNewIdentity() async throws {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true, execsAfterTerm: "node")
        ])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(10))
        #expect(fake.signals.map { "\($0.0):\($0.1)" } == ["4101:terminate", "4101:kill"])
        #expect(!fake.isAlive(4101))
    }

    @Test func aMemberThatExecsJustBeforeItsSignalIsSignalledOnTheNextPass() async throws {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101, execsInsideSignal: "node")])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .seconds(5))
        #expect(fake.signals.map { "\($0.0):\($0.1)" } == ["4101:terminate"])
        #expect(!fake.isAlive(4101))
    }

    @Test func aMemberCaughtMidExecIsRetriedWithoutAnError() async throws {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, unstableIdentityReads: 3), .init(pid: 4102),
        ])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .seconds(5))
        #expect(fake.signals.map(\.0).sorted() == [4101, 4102])
        #expect(!fake.isAlive(4101) && !fake.isAlive(4102))
    }

    @Test func aPIDRecycledBetweenPassesNeverReceivesASignal() async {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true, recyclesAfterTerm: true)
        ])
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(10))
        }
        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(fake.signals.map { "\($0.0):\($0.1)" } == ["4101:terminate"])
        #expect(fake.isAlive(4101))
    }

    @Test func aDeniedSignalIsReportedOnceAndNotRetriedInThePhase() async {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101, denySignalOnce: "denied")])
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(20))
        }
        #expect(error.map { "\($0)" } == "denied")
        #expect(fake.signals.map { "\($0.0):\($0.1)" } == ["4101:kill"])
    }
}
