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

    @Test func phpResolvesConstraintOrFallsBackToNewest() {
        #expect(resolver.php(constraint: "^8.3") == "8.5.8")
        #expect(resolver.php(constraint: "~8.3.0") == "8.3.32")
        #expect(resolver.php(constraint: nil) == "8.5.8")
        #expect(resolver.php(constraint: ">=99") == "8.5.8")
    }
}
