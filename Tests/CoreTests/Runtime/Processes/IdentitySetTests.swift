import Testing

@testable import Core

struct IdentitySetTests {
    private let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101), .init(pid: 4102)])

    @Test func sameSetRejectsDuplicatePIDOnEitherSide() {
        let first = fake.identity(4100)
        let second = fake.identity(4101)
        #expect(!IdentitySet.same([first, second], [first, first]))
        #expect(!IdentitySet.same([first, first], [first, second]))
    }

    @Test func sameSetRejectsExactSetMismatch() {
        let first = fake.identity(4100)
        let second = fake.identity(4101)
        let third = fake.identity(4102)
        #expect(!IdentitySet.same([first, second], [first, third]))
        #expect(IdentitySet.same([second, first], [first, second]))
    }

    @Test func sameSetRejectsAChangedIdentityForTheSamePID() {
        let original = fake.identity(4101)
        fake.setStartID(4101, "recycled")
        #expect(!IdentitySet.same([original], [fake.identity(4101)]))
    }

    @Test func sameSetRejectsNonPositivePIDs() {
        let invalid = ProcessIdentity(pid: 0, pgid: 0, startID: "", executable: "", arguments: "")
        #expect(!IdentitySet.same([invalid], [invalid]))
    }
}
