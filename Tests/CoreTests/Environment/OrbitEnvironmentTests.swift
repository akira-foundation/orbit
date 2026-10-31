import Testing

@testable import Core

@Suite struct OrbitEnvironmentTests {
    @Test func productionValues() {
        let env = OrbitEnvironment.production
        #expect(env.bundleIdentifier == "io.akira.orbit")
        #expect(env.domainSuffix == "orbit.test")
        #expect(env.loopbackAddress == "127.0.0.2")
        #expect(env.helperMachServiceName == "io.akira.orbit.helper")
        #expect(env.agentMachServiceName == "io.akira.orbit.agent")
        #expect(env.dataDirectoryName == "Orbit")
    }

    @Test func developmentValues() {
        let env = OrbitEnvironment.development
        #expect(env.bundleIdentifier == "io.akira.orbit.dev")
        #expect(env.domainSuffix == "orbit-dev.test")
        #expect(env.loopbackAddress == "127.0.0.3")
        #expect(env.helperMachServiceName == "io.akira.orbit.dev.helper")
        #expect(env.agentMachServiceName == "io.akira.orbit.dev.agent")
        #expect(env.dataDirectoryName == "Orbit (Dev)")
    }

    @Test func environmentsDoNotCollide() {
        let production = OrbitEnvironment.production
        let development = OrbitEnvironment.development
        #expect(production.bundleIdentifier != development.bundleIdentifier)
        #expect(production.domainSuffix != development.domainSuffix)
        #expect(production.loopbackAddress != development.loopbackAddress)
        #expect(production.helperMachServiceName != development.helperMachServiceName)
        #expect(production.agentMachServiceName != development.agentMachServiceName)
        #expect(production.dataDirectoryName != development.dataDirectoryName)
    }

    @Test(arguments: [
        ("io.akira.orbit", OrbitEnvironment.production),
        ("io.akira.orbit.dev", .development),
        ("com.example.other", .production),
        ("io.akira.orbit.dev.helper", .production),
        ("io.akira.orbit.development", .production),
    ])
    func currentFromBundleIdentifier(id: String, expected: OrbitEnvironment) {
        #expect(OrbitEnvironment.current(bundleIdentifier: id) == expected)
    }

    @Test func everyEnvironmentResolvesFromItsOwnBundleIdentifier() {
        let resolved = OrbitEnvironment.allCases.map { OrbitEnvironment.current(bundleIdentifier: $0.bundleIdentifier) }
        #expect(resolved == OrbitEnvironment.allCases)
    }

    @Test func currentWithoutBundleIsProduction() {
        #expect(OrbitEnvironment.current(bundleIdentifier: nil) == .production)
    }
}
