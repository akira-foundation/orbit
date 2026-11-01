import Darwin
import Testing

@testable import Core

struct KernelProcessReaderTests {
    @Test(arguments: [(1, "100:000001"), (999_999, "100:999999"), (120_000, "100:120000")])
    func startIDPadsMicrosecondsToSixDigits(micros: Int32, expected: String) {
        #expect(KernelProcessReader.startID(timeval(tv_sec: 100, tv_usec: micros)) == expected)
    }
}
