import Testing

@testable import Core

struct RuntimeErrorsTests {
    @Test func descriptionsMatchTheGoMessages() {
        #expect(RuntimeError.processGone.description == "process already finished")
        #expect(RuntimeError.identityMismatch("").description == "runtime: process identity mismatch")
        #expect(RuntimeError.identityMismatch("PID 4101").description == "runtime: process identity mismatch: PID 4101")
        #expect(RuntimeError.groupNotEmpty.description == "runtime: process group is not empty")
        #expect(
            RuntimeError.unownedEndpoint(nil).description
                == "runtime: endpoint listener is not owned by supervised group")
        #expect(
            RuntimeError.unownedEndpoint("changed").description
                == "runtime: endpoint listener is not owned by supervised group: changed")
        #expect(RuntimeError.unexpectedSupervisorExit.description == "runtime: supervisor exited unexpectedly")
        #expect(RuntimeError.cancelled.description == "cancelled")
        #expect(RuntimeError.message("as is").description == "as is")
    }

    @Test func joinReturnsNilWithoutErrors() {
        #expect(RuntimeErrors.join([nil, nil]) == nil)
        #expect(RuntimeErrors.join([]) == nil)
    }

    @Test func joinReturnsTheOnlyErrorItself() {
        let joined = RuntimeErrors.join([nil, RuntimeError.groupNotEmpty, nil])
        #expect(joined as? RuntimeError == .groupNotEmpty)
    }

    @Test func joinFlattensNestedAggregates() throws {
        let inner = try #require(RuntimeErrors.join([RuntimeError.message("a"), RuntimeError.message("b")]))
        let joined = try #require(RuntimeErrors.join([inner, RuntimeError.message("c")]) as? RuntimeErrors)
        #expect(joined.errors.count == 3)
        #expect(joined.description == "a\nb\nc")
    }

    @Test func containsMatchesDetailedCasesByKind() {
        let error: any Error = RuntimeError.identityMismatch("PID 4101")
        #expect(error.contains(.identityMismatch("")))
        #expect(!error.contains(.groupNotEmpty))
        #expect(RuntimeError.unownedEndpoint("changed").contains(.unownedEndpoint(nil)))
        #expect(!RuntimeError.message("a").contains(.message("b")))
    }

    @Test func containsLooksInsideAggregates() throws {
        let joined = try #require(RuntimeErrors.join([RuntimeError.cancelled, RuntimeError.message("a")]))
        #expect(joined.contains(.cancelled))
        #expect(joined.contains(.message("a")))
        #expect(!joined.contains(.processGone))
    }

    @Test func signalsCarryTheDarwinNumbers() {
        #expect(ProcessSignal.terminate.rawValue == 15)
        #expect(ProcessSignal.kill.rawValue == 9)
        #expect(ProcessSignal.custom(2).rawValue == 2)
    }
}
