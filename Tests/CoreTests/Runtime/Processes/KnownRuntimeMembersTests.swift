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
}
