import Testing

@testable import Core

struct KnownRuntimeMembersTests {
    private let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101)])

    @Test func combineRejectsNonPositivePID() {
        let invalid = ProcessIdentity(
            pid: 0, pgid: 4100, startID: "start-0", executable: "/fixture/process", arguments: "")
        #expect(throws: RuntimeError.identityMismatch("invalid PID 0")) {
            try KnownRuntimeMembers.combine([fake.identity(4100), invalid])
        }
    }

    @Test func combineCountsConflictingSnapshotsAsUnsettled() throws {
        let original = fake.identity(4101)
        fake.setStartID(4101, "recycled")
        let combined = try KnownRuntimeMembers.combine([fake.identity(4100), original, fake.identity(4101), original])
        #expect(combined == ProcessListing(identities: [fake.identity(4100)], unsettled: [4101]))
    }

    @Test func combineMergesRepeatedSnapshotsInPIDOrder() throws {
        let combined = try KnownRuntimeMembers.combine([fake.identity(4101), fake.identity(4100), fake.identity(4101)])
        #expect(combined == ProcessListing(identities: [fake.identity(4100), fake.identity(4101)], unsettled: []))
    }

    @Test func aConflictAcrossTheTwoListingsLeavesTheMemberUnsettled() throws {
        let inspector = FakeProcessInspector.supervised(members: [.init(pid: 4101, conflictingForDescendantReads: 1)])
        let members = try KnownRuntimeMembers(supervisor: inspector.identity(4100), inspector: inspector)
        #expect(members.ordered == [inspector.identity(4100)])
        #expect(members.unsettled == [4101])
        #expect(!members.containsOnly(4100))
    }

    @Test func refreshAdoptsTheIdentityOfAMemberThatExecuted() throws {
        let inspector = FakeProcessInspector.supervised(members: [
            .init(pid: 4101, ignoresTerm: true, execsAfterTerm: "node")
        ])
        var members = try KnownRuntimeMembers(supervisor: inspector.identity(4100), inspector: inspector)
        let original = inspector.identity(4101)
        try inspector.signal(original, .terminate)
        try members.refresh(supervisor: inspector.identity(4100), inspector: inspector)
        #expect(members.members[4101]?.executable == "node")
        #expect(members.members[4101]?.startID == original.startID)
    }

    @Test func refreshRejectsAPIDThatNowBelongsToAnotherProcess() throws {
        let inspector = FakeProcessInspector.supervised(members: [.init(pid: 4101)])
        var members = try KnownRuntimeMembers(supervisor: inspector.identity(4100), inspector: inspector)
        inspector.setStartID(4101, "recycled")
        #expect(throws: RuntimeError.identityMismatch("PID 4101 changed while cleaning up")) {
            try members.refresh(supervisor: inspector.identity(4100), inspector: inspector)
        }
    }

    @Test func refreshKeepsAMemberCaughtMidExecAsUnsettled() throws {
        let inspector = FakeProcessInspector.supervised(members: [.init(pid: 4101)])
        var members = try KnownRuntimeMembers(supervisor: inspector.identity(4100), inspector: inspector)
        inspector.add(.init(pid: 4101, unstableIdentityReads: 1))
        try members.refresh(supervisor: inspector.identity(4100), inspector: inspector)
        #expect(members.unsettled == [4101])
        #expect(members.members[4101] != nil)
    }
}
