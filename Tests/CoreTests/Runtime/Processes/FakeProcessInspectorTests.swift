import Testing

@testable import Core

struct FakeProcessInspectorTests {
    @Test func fakeDescendantsFollowParents() throws {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101), .init(pid: 4102, ppid: 4101)])
        fake.add(.init(pid: 9000, pgid: 9000))
        let descendants = try fake.descendants(of: 4100)
        #expect(descendants == [fake.identity(4101), fake.identity(4102)])
    }
}
