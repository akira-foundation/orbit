import Darwin
import Foundation
import Testing

@testable import Core

struct PipeDrainTests {
    @Test func readsBothPipesToTheEndWhenTheSecondFillsBeforeTheFirstCloses() throws {
        let process = try PosixSpawner.spawn(
            SpawnRequest(
                executable: "/bin/sh",
                arguments: ["-c", "/usr/bin/head -c 200000 /dev/zero >&2; /usr/bin/head -c 100000 /dev/zero"],
                environment: EnvironmentList([]), captureOutput: true))
        let output = try #require(process.stdout)
        let errors = try #require(process.stderr)
        let captured = PipeDrain.readToEnd([output.fileDescriptor, errors.fileDescriptor])
        #expect(captured.map(\.count) == [100_000, 200_000])
        #expect(process.waitForExit() == .exited(0))
    }

    @Test func returnsNothingForNoDescriptors() {
        #expect(PipeDrain.readToEnd([]) == [])
    }
}
