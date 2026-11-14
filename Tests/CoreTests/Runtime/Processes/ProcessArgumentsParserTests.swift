import Testing

@testable import Core

struct ProcessArgumentsParserTests {
    private let malformed = RuntimeError.message("inspect process: malformed arguments")

    @Test func argumentsParserJoinsArgv() throws {
        let buffer = Self.argc(3) + Array("/bin/sh\0\0\0sh\0-c\0echo ready\0PATH=/x\0".utf8)
        #expect(try ProcessArgumentsParser.parse(buffer) == "sh -c echo ready")
    }

    @Test func argumentsParserRejectsTruncatedBuffer() {
        #expect(throws: malformed) {
            try ProcessArgumentsParser.parse([3, 0])
        }
    }

    @Test func argumentsParserRejectsMissingArguments() {
        let buffer = Self.argc(3) + Array("/bin/sh\0\0sh\0-c\0".utf8)
        #expect(throws: malformed) {
            try ProcessArgumentsParser.parse(buffer)
        }
    }

    @Test func argumentsParserRejectsUnterminatedArgument() {
        let buffer = Self.argc(1) + Array("/bin/sh\0sh".utf8)
        #expect(throws: malformed) {
            try ProcessArgumentsParser.parse(buffer)
        }
    }

    private static func argc(_ count: Int32) -> [UInt8] {
        withUnsafeBytes(of: count) { Array($0) }
    }
}
