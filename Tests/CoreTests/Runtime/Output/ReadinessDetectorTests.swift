import Testing

@testable import Core

struct ReadinessDetectorTests {
    @Test func thereAreThirteenPhrases() {
        #expect(ReadinessDetector.phrases.count == 13)
    }

    @Test(arguments: ReadinessDetector.phrases)
    func everyPhraseLooksReady(phrase: String) {
        #expect(ReadinessDetector.looksReady(phrase.uppercased()))
        #expect(ReadinessDetector.looksReady("\u{1B}[32m" + phrase + "\u{1B}[39m"))
        #expect(ReadinessDetector.looksReady("  ->  " + phrase + " extra"))
    }

    @Test func plainOutputIsNotReady() {
        #expect(!ReadinessDetector.looksReady("compiling..."))
        #expect(!ReadinessDetector.looksReady(""))
    }
}
