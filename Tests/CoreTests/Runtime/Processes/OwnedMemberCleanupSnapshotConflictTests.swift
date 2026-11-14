import Testing

@testable import Core

struct OwnedMemberCleanupSnapshotConflictTests {
    @Test func conflictInTheFirstListingDoesNotAbortCleanup() async throws {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, conflictingForDescendantReads: 1), .init(pid: 4102),
        ])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .seconds(5))
        #expect(fake.signals.map { "\($0.0):\($0.1)" } == ["4102:terminate", "4101:terminate"])
        #expect(!fake.isAlive(4101) && !fake.isAlive(4102))
    }

    @Test func conflictThatOutlastsTheTermPhaseReceivesOnlyKillOnceSettled() async throws {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true), .init(pid: 4102, ignoresTerm: true, conflictingForDescendantReads: 3),
        ])
        try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .zero)
        #expect(fake.signals.map { "\($0.0):\($0.1)" } == ["4101:terminate", "4101:kill", "4102:kill"])
        #expect(!fake.isAlive(4101) && !fake.isAlive(4102))
    }

    @Test func conflictThatNeverSettlesEndsInErrorWithoutSignallingTheMember() async {
        let fake = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, conflictingForDescendantReads: .max), .init(pid: 4102),
        ])
        let error = await #expect(throws: (any Error).self) {
            try await OwnedMemberCleanup.run(supervisor: fake.identity(4100), inspector: fake, grace: .milliseconds(10))
        }
        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(error.map { "\($0)" }?.contains("members did not settle: 4101") == true)
        #expect(fake.signals.map { "\($0.0):\($0.1)" } == ["4102:terminate"])
        #expect(fake.isAlive(4101))
    }
}
