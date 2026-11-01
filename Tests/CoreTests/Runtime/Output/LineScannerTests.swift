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

    @Test func scannerAcceptsALineWhoseTerminatorFitsTheMaximum() {
        var scanner = LineScanner()
        let line = String(repeating: "a", count: LineScanner.maximumLineLength - 1)
        #expect(scanner.feed(Array((line + "\n").utf8)) == [line])
        #expect(!scanner.overflowed)
    }

    @Test func scannerRejectsALineWhoseTerminatorDoesNotFitTheMaximum() {
        var scanner = LineScanner()
        let bytes = [UInt8](repeating: 0x61, count: LineScanner.maximumLineLength) + [0x0A]
        #expect(scanner.feed(bytes) == [])
        #expect(scanner.overflowed)
    }

    @Test func scannerCountsTheCarriageReturnAgainstTheMaximum() {
        var scanner = LineScanner()
        let bytes = [UInt8](repeating: 0x61, count: LineScanner.maximumLineLength - 1) + [0x0D, 0x0A]
        #expect(scanner.feed(bytes) == [])
        #expect(scanner.overflowed)
    }

    @Test func scannerRejectsAnUnterminatedLastLineOfTheMaximum() {
        var scanner = LineScanner()
        #expect(scanner.feed([UInt8](repeating: 0x61, count: LineScanner.maximumLineLength)) == [])
        #expect(scanner.finish() == [])
        #expect(scanner.overflowed)
    }

    @Test func scannerAcceptsAnUnterminatedLastLineJustBelowTheMaximum() {
        var scanner = LineScanner()
        let line = String(repeating: "a", count: LineScanner.maximumLineLength - 1)
        #expect(scanner.feed(Array(line.utf8)) == [])
        #expect(scanner.finish() == [line])
    }

    @Test func scannerJoinsACarriageReturnLineFeedSplitAcrossFeeds() {
        var scanner = LineScanner()
        #expect(scanner.feed(Array("a\r".utf8)) == [])
        #expect(scanner.feed(Array("\nb\n".utf8)) == ["a", "b"])
    }

    @Test func scannerStripsALoneTrailingCarriageReturnOnFinish() {
        var scanner = LineScanner()
        #expect(scanner.feed(Array("x\r".utf8)) == [])
        #expect(scanner.finish() == ["x"])
        #expect(scanner.feed([0x0D]) == [])
        #expect(scanner.finish() == [""])
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
