import Testing

@testable import Core

struct RestartPolicyTests {
    @Test(arguments: [1, 2, 3, 4, 5, 6])
    func restartDelays(attempt: Int) {
        let expected: [Int: Duration] = [
            1: .seconds(1), 2: .seconds(2), 3: .seconds(4), 4: .seconds(8), 5: .seconds(16),
        ]

        #expect(RestartPolicy.delay(forAttempt: attempt) == expected[attempt])
    }

    @Test func limitsMatchTheGoReference() {
        #expect(RestartPolicy.maximumAttempts == 5)
        #expect(RestartPolicy.maximumDelay == .seconds(16))
    }
}
