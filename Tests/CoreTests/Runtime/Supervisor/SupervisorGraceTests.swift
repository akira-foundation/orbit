import Testing

@testable import Core

struct SupervisorGraceTests {
    @Test(arguments: [nil, "0", "-5", "abc", "", "20ms"])
    func graceFallsBackForMissingOrInvalidValues(value: String?) {
        #expect(SupervisorGrace.parse(value) == .seconds(5))
    }

    @Test func graceParsesPositiveMilliseconds() {
        #expect(SupervisorGrace.parse("20") == .milliseconds(20))
    }

    @Test func environmentReplacesEveryPreviousGraceEntry() {
        let base = EnvironmentList([
            "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=1", "A=b", "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=2",
        ])
        #expect(
            SupervisorGrace.environment(base, grace: .zero).entries == [
                "A=b", "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=5000",
            ])
        #expect(
            SupervisorGrace.environment(base, grace: .milliseconds(20)).entries == [
                "A=b", "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=20",
            ])
        #expect(
            SupervisorGrace.environment(base, grace: .seconds(-1)).entries == [
                "A=b", "ORBIT_RUNTIME_SUPERVISOR_GRACE_MS=5000",
            ])
    }
}
