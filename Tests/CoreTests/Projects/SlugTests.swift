import Testing

@testable import Core

@Suite struct SlugTests {
    @Test(arguments: [
        ("nosferry.com", "nosferry.com"), ("nosferry/nosferry.com", "nosferry-nosferry.com"), ("   ", "project"),
        ("My App", "my-app"), ("Café", "cafe"), ("@acme/web", "acme-web"), ("..hidden..", "hidden"),
        ("a__b", "a-b"),
    ])
    func slugCases(input: String, expected: String) {
        #expect(Slug.make(from: input) == expected)
    }
}
