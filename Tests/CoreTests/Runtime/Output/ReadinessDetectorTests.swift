import Testing

@testable import Core

struct ReadinessDetectorTests {
    @Test func thereAreThirteenPhrases() {
        #expect(ReadinessDetector.phrases.count == 13)
    }

    @Test(arguments: [
        "ready in", "ready - started", "local:", "localhost:", "listening on", "server running", "server started",
        "compiled successfully", "started development server", "uvicorn running on", "application startup complete",
        "starting development server at", "running on http",
    ])
    func everyPhraseLooksReady(phrase: String) {
        #expect(ReadinessDetector.looksReady(phrase.uppercased()))
        #expect(ReadinessDetector.looksReady("\u{1B}[32m" + phrase + "\u{1B}[39m"))
        #expect(ReadinessDetector.looksReady("  ->  " + phrase + " extra"))
    }

    @Test func readinessSurvivesACombiningMarkAfterThePhrase() {
        #expect(ReadinessDetector.looksReady("local:\u{301}"))
    }

    @Test func plainOutputIsNotReady() {
        #expect(!ReadinessDetector.looksReady("compiling..."))
        #expect(!ReadinessDetector.looksReady(""))
    }
}
