import Testing

@testable import Core

@Suite struct SlugTests {
    @Test(arguments: [
        ("nosferry.com", "nosferry.com"), ("nosferry/nosferry.com", "nosferry-nosferry.com"), ("   ", "project"),
        ("My App", "my-app"), ("Café", "cafe"), ("@acme/web", "acme-web"), ("..hidden..", "hidden"),
        ("a__b", "a-b"), ("a. b", "a.b"), ("a..b", "a.b"), ("-a-.-b-", "a.b"), (".", "project"),
    ])
    func slugCases(input: String, expected: String) {
        #expect(Slug.make(from: input) == expected)
    }

    @Test func longNamesYieldLabelsOfAtMost63CharactersWithoutTrailingHyphen() {
        let slug = Slug.make(from: String(repeating: "a", count: 62) + " " + String(repeating: "b", count: 8))

        let labels = slug.split(separator: ".")
        #expect(labels.allSatisfy { $0.count <= 63 && !$0.hasSuffix("-") && !$0.hasPrefix("-") })
        #expect(!slug.isEmpty)
    }
}
