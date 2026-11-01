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

    @Test func combineRejectsConflictingSnapshotsForOnePID() {
        let original = fake.identity(4101)
        fake.setStartID(4101, "recycled")
        #expect(throws: RuntimeError.identityMismatch("conflicting snapshots for PID 4101")) {
            try KnownRuntimeMembers.combine([original, fake.identity(4101)])
        }
    }

    @Test func combineMergesRepeatedSnapshotsInPIDOrder() throws {
        let combined = try KnownRuntimeMembers.combine([fake.identity(4101), fake.identity(4100), fake.identity(4101)])
        #expect(combined == [fake.identity(4100), fake.identity(4101)])
    }
}
