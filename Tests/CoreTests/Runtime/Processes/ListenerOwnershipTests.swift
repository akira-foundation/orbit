import Testing

@testable import Core

struct ListenerOwnershipTests {
    private let endpoint = Endpoint(host: "127.0.0.1", port: 5173)

    @Test func verifyRejectsUnrelatedProcess() {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101)])
        fake.add(.init(pid: 9000, pgid: 9000))
        fake.listenerReads = [[9000], [9000]]
        #expect(throws: RuntimeError.unownedEndpoint(nil)) {
            try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        }
    }

    @Test func verifyRejectsMixedOwners() {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101)])
        fake.add(.init(pid: 9000, pgid: 9000))
        fake.listenerReads = [[4101, 9000], [4101, 9000]]
        #expect(throws: RuntimeError.unownedEndpoint(nil)) {
            try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        }
    }

    @Test func verifyRejectsAnUnsettledOwner() {
        let fake = FakeProcessInspector.supervised()
        fake.add(.init(pid: 9000, pgid: 9000))
        fake.unsettled = [9000]
        fake.listenerReads = [[4100, 9000], [4100, 9000]]
        #expect(throws: RuntimeError.unownedEndpoint("listener owner did not settle during inspection")) {
            try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        }
    }

    @Test func verifyRejectsAnOwnerThatIsUnsettledInTheConfirmingRead() {
        let fake = FakeProcessInspector.supervised()
        fake.add(.init(pid: 9000, pgid: 9000))
        fake.unsettled = [9000]
        fake.listenerReads = [[4100], [4100, 9000]]
        #expect(throws: RuntimeError.unownedEndpoint("listener owner set changed during inspection")) {
            try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        }
    }

    @Test func verifyRejectsNoOwners() {
        let fake = FakeProcessInspector.supervised()
        #expect(throws: RuntimeError.unownedEndpoint(nil)) {
            try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        }
    }

    @Test func verifyAcceptsEveryGroupOwner() throws {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101)])
        fake.listenerReads = [[4100, 4101], [4100, 4101]]
        try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        #expect(fake.listenerReads.isEmpty)
    }

    @Test func verifyRejectsOwnerSetChangedDuringInspection() {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101), .init(pid: 4102)])
        fake.listenerReads = [[4101], [4102]]
        #expect(throws: RuntimeError.unownedEndpoint("listener owner set changed during inspection")) {
            try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        }
    }

    @Test func verifyRejectsRecycledSupervisorBeforeReadingOwners() {
        let fake = FakeProcessInspector.supervised(members: [.init(pid: 4101)])
        let expected = fake.identity(4100)
        fake.setStartID(4100, "different-start")
        fake.listenerReads = [[4101], [4101]]
        let error = #expect(throws: RuntimeError.self) {
            try ListenerOwnership.verify(supervisor: expected, endpoint: endpoint, inspector: fake)
        }
        #expect(error?.contains(.identityMismatch("")) == true)
        #expect(fake.listenerReads.count == 2)
    }

    @Test func verifyReturnsInspectorError() {
        let fake = FakeProcessInspector.supervised()
        fake.listenerError = "listener inspection denied"
        let error = #expect(throws: (any Error).self) {
            try ListenerOwnership.verify(supervisor: fake.identity(4100), endpoint: endpoint, inspector: fake)
        }
        #expect(error.map { "\($0)" } == "listener inspection denied")
    }
}
