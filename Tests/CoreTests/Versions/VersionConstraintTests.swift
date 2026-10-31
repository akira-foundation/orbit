import Testing

@testable import Core

@Suite struct VersionConstraintTests {
    @Test(
        arguments: [
            ("^22.0.0", "22.23.1"), ("^20.0.0", "20.20.2"), ("~20.20.0", "20.20.2"), ("~20.19.0", nil),
            (">=20.0.0", "24.18.0"), (">=20.0.0 <24.0.0", "22.23.1"), ("20.x", "20.20.2"), ("20", "20.20.2"),
            ("20.20.2", "20.20.2"), ("20.20.3", nil), ("^18.0.0|^22.0.0", "22.23.1"), ("^18.0.0 || ^20.0.0", "20.20.2"),
            ("garbage-constraint", nil), ("", nil),
        ] as [(String, String?)])
    func resolve(constraint: String, expected: String?) {
        #expect(VersionConstraint.resolve(constraint, among: ["24.18.0", "22.23.1", "20.20.2"]) == expected)
    }

    @Test(
        arguments: [
            ("8.3.17", SemanticVersion(8, 3, 17)), ("v8.3.17", SemanticVersion(8, 3, 17)),
            ("8.3", SemanticVersion(8, 3, 0)), ("8", SemanticVersion(8, 0, 0)), ("not-a-version", nil),
        ] as [(String, SemanticVersion?)])
    func parse(input: String, expected: SemanticVersion?) {
        #expect(SemanticVersion(input) == expected)
    }

    @Test func bundledCatalogMatchesTheGoReference() {
        #expect(RuntimeCatalog.bundled.node == ["24.18.0", "22.23.1", "20.20.2"])
        #expect(RuntimeCatalog.bundled.php == ["8.5.8", "8.4.23", "8.3.32"])
        #expect(RuntimeCatalog.bundled.python == ["3.13.14", "3.12.13", "3.11.15"])
    }
}
