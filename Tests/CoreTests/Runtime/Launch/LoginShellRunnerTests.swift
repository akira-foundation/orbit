import Foundation
import Testing

@testable import Core

struct LoginShellRunnerTests {
    @Test func returnsStandardOutput() async {
        let output = await LoginShellRunner().run(shell: "/bin/sh", script: "echo hi", timeout: .seconds(5))
        #expect(output == "hi\n")
    }

    @Test func returnsNilAndStopsWaitingAfterTheTimeout() async {
        let started = ContinuousClock.now
        let output = await LoginShellRunner().run(shell: "/bin/sh", script: "sleep 5", timeout: .milliseconds(100))
        #expect(output == nil)
        #expect(ContinuousClock.now - started < .seconds(2))
    }

    @Test func returnsNilWhenTheShellDoesNotExist() async {
        let output = await LoginShellRunner().run(shell: "/nonexistent/sh", script: "echo hi", timeout: .seconds(1))
        #expect(output == nil)
    }
}
