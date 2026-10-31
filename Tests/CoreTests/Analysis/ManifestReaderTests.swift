import Testing

@testable import Core

@Suite struct ManifestReaderTests {
    @Test(arguments: [#"{"a":"x,}"}"#, #"{"a":"x\",}"}"#, #"{"a":["b,]"]}"#])
    func aCommaInsideAStringIsNotATrailingComma(manifest: String) throws {
        let dir = try TemporaryDirectory()
        try dir.write(manifest, to: "package.json")

        #expect(try ManifestReader.package(in: dir.path) != nil)
    }

    @Test func aTrailingCommaBeforeAClosingBracketIsRejected() throws {
        let dir = try TemporaryDirectory()
        try dir.write(#"{"a":[1,]}"#, to: "package.json")

        #expect(throws: AnalysisError.invalidManifest(path: dir.path + "/package.json", reason: "trailing comma")) {
            try ManifestReader.package(in: dir.path)
        }
    }
}
