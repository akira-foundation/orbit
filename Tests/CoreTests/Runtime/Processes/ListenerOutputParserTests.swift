import Testing

@testable import Core

struct ListenerOutputParserTests {
    @Test func listenerParserReturnsEveryUniquePID() throws {
        #expect(try ListenerOutputParser.processIDs("p4101\np4100\np4101\n") == [4100, 4101])
    }

    @Test func listenerParserSkipsFileDescriptorFields() throws {
        #expect(try ListenerOutputParser.processIDs("p3025\nf14\n") == [3025])
    }

    @Test func listenerParserAcceptsMultipleOwnersWithFields() throws {
        #expect(try ListenerOutputParser.processIDs("p3025\nf14\np4100\nf20\n") == [3025, 4100])
    }

    @Test(arguments: ["p4100\nwarning\np4101\n", "p4100 \n", "p3025\nz9\n", "\n", "p\n", "p0\n"])
    func listenerParserRejectsAmbiguousOutput(output: String) {
        #expect(throws: RuntimeError.self) {
            try ListenerOutputParser.processIDs(output)
        }
    }

    @Test func listenerParserNamesTheEmptyField() {
        #expect(throws: RuntimeError.message("ambiguous empty lsof field")) {
            try ListenerOutputParser.processIDs("\n")
        }
    }

    @Test func listenerParserEmptyOutputIsNoOwners() throws {
        #expect(try ListenerOutputParser.processIDs("") == [])
    }
}
