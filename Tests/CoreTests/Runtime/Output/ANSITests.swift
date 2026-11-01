import Testing

@testable import Core

struct ANSITests {
    @Test func ansiStripLeavesPlainTextAndRemovesSequences() {
        #expect(ANSI.strip("plain") == "plain")
        #expect(ANSI.strip("\u{1B}[1;32mgreen\u{1B}[0m") == "green")
    }
}
