import Testing

@testable import Core

struct LineScannerTests {
    @Test func scannerSplitsAcrossFeedsAndStripsCarriageReturn() {
        var scanner = LineScanner()
        #expect(scanner.feed(Array("ab".utf8)) == [])
        #expect(scanner.feed(Array("c\r\nd\n\n".utf8)) == ["abc", "d", ""])
    }

    @Test func scannerDeliversUnterminatedLastLine() {
        var scanner = LineScanner()
        #expect(scanner.feed(Array("tail".utf8)) == [])
        #expect(scanner.finish() == ["tail"])
        #expect(scanner.finish() == [])
    }

    @Test func scannerStopsAtOneMebibyteLine() {
        var scanner = LineScanner()
        #expect(scanner.feed([UInt8](repeating: 0x61, count: LineScanner.maximumLineLength + 1)) == [])
        #expect(scanner.overflowed)
        #expect(scanner.feed(Array("x\n".utf8)) == [])
        #expect(scanner.finish() == [])
    }

    @Test func scannerKeepsALineOfExactlyTheMaximum() {
        var scanner = LineScanner()
        let line = String(repeating: "a", count: LineScanner.maximumLineLength)
        #expect(scanner.feed(Array((line + "\n").utf8)) == [line])
        #expect(!scanner.overflowed)
    }

    @Test func scannerDeliversLinesBeforeTheOverflow() {
        var scanner = LineScanner()
        var bytes = Array("ok\n".utf8)
        bytes.append(contentsOf: [UInt8](repeating: 0x61, count: LineScanner.maximumLineLength + 1))
        #expect(scanner.feed(bytes) == ["ok"])
        #expect(scanner.overflowed)
    }

    @Test func scannerReplacesInvalidUTF8() {
        var scanner = LineScanner()
        #expect(scanner.feed([0x66, 0xFF, 0x0A]) == ["f\u{FFFD}"])
    }
}
