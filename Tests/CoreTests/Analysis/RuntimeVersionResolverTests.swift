import Testing

@testable import Core

struct RuntimeVersionResolverTests {
    private let resolver = RuntimeVersionResolver(catalog: .bundled)
    private let newest = RuntimeCatalog.bundled.node[0]

    @Test func nvmrcTakesPriority() throws {
        let dir = try TemporaryDirectory()
        try dir.write("v20\n", to: ".nvmrc")
        try dir.write("22", to: ".node-version")

        #expect(resolver.node(in: dir.path, engines: nil) == "20.20.2")
    }

    @Test func nodeVersionFileWhenNoNvmrc() throws {
        let dir = try TemporaryDirectory()
        try dir.write("22.23.1", to: ".node-version")

        #expect(resolver.node(in: dir.path, engines: nil) == "22.23.1")
    }

    @Test func enginesFieldWhenNoVersionFile() throws {
        let dir = try TemporaryDirectory()

        #expect(resolver.node(in: dir.path, engines: ">=20.0.0 <22.0.0") == "20.20.2")
    }

    @Test func noSignalFallsBackToNewest() throws {
        let dir = try TemporaryDirectory()

        #expect(resolver.node(in: dir.path, engines: nil) == newest)
    }

    @Test func unparseableFallsBackToNewest() throws {
        let dir = try TemporaryDirectory()

        #expect(resolver.node(in: dir.path, engines: "not-a-real-constraint") == newest)
    }

    @Test func unsatisfiableConstraintFallsBackToNewest() throws {
        let dir = try TemporaryDirectory()

        #expect(resolver.node(in: dir.path, engines: ">=99.0.0") == newest)
    }

    @Test func anExactNodeVersionOutsideTheCatalogKeepsItsMajor() throws {
        let dir = try TemporaryDirectory()
        try dir.write("v20.11.0\n", to: ".nvmrc")

        #expect(resolver.node(in: dir.path, engines: nil) == "20.20.2")
    }

    @Test func anExactEnginesVersionOutsideTheCatalogKeepsItsMajor() throws {
        let dir = try TemporaryDirectory()

        #expect(resolver.node(in: dir.path, engines: "20.11.0") == "20.20.2")
        #expect(resolver.node(in: dir.path, engines: "v22.1.0") == "22.23.1")
    }

    @Test func anExactNodeVersionWithAnUnknownMajorFallsBackToNewest() throws {
        let dir = try TemporaryDirectory()
        try dir.write("99\n", to: ".nvmrc")

        #expect(resolver.node(in: dir.path, engines: nil) == newest)
    }

    @Test func anExactPHPVersionOutsideTheCatalogKeepsItsMinorThenItsMajor() {
        let catalog = RuntimeCatalog(node: [], php: ["9.0.1", "8.5.8", "8.3.32"], python: [])
        let sparse = RuntimeVersionResolver(catalog: catalog)

        #expect(resolver.php(constraint: "8.4.1") == "8.4.23")
        #expect(sparse.php(constraint: "8.4.1") == "8.5.8")
        #expect(resolver.php(constraint: "7.4.33") == "8.5.8")
    }

    @Test func phpResolvesConstraintOrFallsBackToNewest() {
        #expect(resolver.php(constraint: "^8.3") == "8.5.8")
        #expect(resolver.php(constraint: "~8.3.0") == "8.3.32")
        #expect(resolver.php(constraint: nil) == "8.5.8")
        #expect(resolver.php(constraint: ">=99") == "8.5.8")
    }
}
