import Testing

@testable import Core

@Suite struct ProjectRegistryTests {
    private func registry(
        environment: OrbitEnvironment = .production, analyzer: any ProjectAnalyzing = FixedAnalyzer()
    ) throws -> (ProjectRegistry, Store) {
        let store = try Store.inMemory()
        return (ProjectRegistry(store: store, analyzer: analyzer, environment: environment), store)
    }

    @Test func addingTheSamePathReturnsTheSameProject() async throws {
        let (registry, store) = try registry()

        let first = try await registry.add(path: "/a/shop", overwrite: false)
        let again = try await registry.add(path: "/a/shop", overwrite: false)

        #expect(again.id == first.id)
        #expect(try await store.projects().count == 1)
    }

    @Test func addingTheSameSlugAtAnotherPathConflicts() async throws {
        let (registry, _) = try registry()
        _ = try await registry.add(path: "/a/shop", overwrite: false)

        await #expect(throws: ProjectConflict(existingName: "shop", slug: "shop")) {
            try await registry.add(path: "/b/shop", overwrite: false)
        }
    }

    @Test func overwriteReplacesTheSlugOwner() async throws {
        let (registry, store) = try registry()
        let old = try await registry.add(path: "/a/shop", overwrite: false)

        let fresh = try await registry.add(path: "/b/shop", overwrite: true)

        let listed = try await store.projects()
        #expect(fresh.id != old.id)
        #expect(listed.map(\.path) == ["/b/shop"])
    }

    @Test func overwriteKeepsNoGroupMembershipOfTheReplacedProject() async throws {
        let (registry, store) = try registry()
        let old = try await registry.add(path: "/a/shop", overwrite: false)
        let group = try await store.createGroup(named: "stack")
        try await store.addProject(old.id, toGroup: group.id)

        _ = try await registry.add(path: "/b/shop", overwrite: true)

        #expect(try await store.groups().first { $0.id == group.id }?.projectIDs == [])
    }

    @Test func addingTheSameFolderThroughAnotherSpellingReturnsTheSameProject() async throws {
        let directory = try TemporaryDirectory()
        let shop = try directory.makeDirectory("shop")
        try directory.write(#"{"name":"shop","dependencies":{"@nestjs/core":"^10"}}"#, to: "shop/package.json")
        let (registry, store) = try registry(analyzer: ProjectAnalyzer())

        let first = try await registry.add(path: shop, overwrite: false)
        let again = try await registry.add(path: shop + "/", overwrite: false)

        #expect(again.id == first.id)
        #expect(try await store.projects().count == 1)
    }

    @Test func anAmbiguousMonorepoIsRejectedAndNothingIsStored() async throws {
        let directory = try TemporaryDirectory()
        let laravel = #"{"require":{"laravel/framework":"^11.0"}}"#
        try directory.write(laravel, to: "stack/api/composer.json")
        try directory.write(laravel, to: "stack/admin/composer.json")
        let (registry, store) = try registry(analyzer: ProjectAnalyzer())
        let path = CanonicalPath.of("\(directory.path)/stack")

        await #expect(throws: AnalysisError.ambiguousLayout(path: path, reason: MonorepoAnalyzer.ambiguousReason)) {
            try await registry.add(path: "\(directory.path)/stack", overwrite: false)
        }
        #expect(try await store.projects().isEmpty)
    }

    @Test func domainUsesTheEnvironmentSuffix() throws {
        let (production, _) = try registry(environment: .production)
        let (development, _) = try registry(environment: .development)

        #expect(production.domain(forSlug: "shop") == "shop.orbit.test")
        #expect(development.domain(forSlug: "shop") == "shop.orbit-dev.test")
    }

    @Test func addedProjectCarriesTheDomainAndParkedFolder() async throws {
        let (registry, _) = try registry(environment: .development)

        let project = try await registry.add(path: "/a/shop", overwrite: false, parkedFolder: "/a")

        #expect(project.localDomain == "shop.orbit-dev.test")
        #expect(project.parkedFolder == "/a")
    }

    @Test func emptyPathIsRejectedBeforeTheAnalyzerRuns() async throws {
        let (registry, _) = try registry()

        await #expect(throws: AnalysisError.emptyPath) {
            try await registry.add(path: "   ", overwrite: false)
        }
    }

    private func duplicateError(inserting project: Project, into store: Store) async -> StoreError? {
        do {
            try await store.upsert(project)
            return nil
        } catch {
            return error as? StoreError
        }
    }

    @Test func aDomainTakenBehindTheRegistrysBackSurfacesAsDuplicate() async throws {
        let (registry, store) = try registry()
        try await store.upsert(
            Project(
                analysis: ProjectAnalysis(name: "other", path: "/c/other"), slug: "other",
                localDomain: "shop.orbit.test"))

        await #expect(throws: StoreError.duplicate(column: "local_domain")) {
            try await registry.add(path: "/a/shop", overwrite: false)
        }
    }

    @Test func competingRegistrationOfTheSameSlugSurfacesAsConflict() async throws {
        let (registry, store) = try registry()
        let competitor = ProjectAnalysis(name: "shop", path: "/c/shop")
        try await store.upsert(Project(analysis: competitor, slug: "shop", localDomain: "shop.orbit.test"))
        let late = Project(
            analysis: ProjectAnalysis(name: "shop", path: "/d/shop"), slug: "shop", localDomain: "shop.orbit.test")
        let error = try #require(await duplicateError(inserting: late, into: store))

        await #expect(throws: ProjectConflict(existingName: "shop", slug: "shop")) {
            try await registry.resolveRace(late, after: error)
        }
    }

    @Test func competingRegistrationOfTheSamePathReturnsTheWinner() async throws {
        let (registry, store) = try registry()
        let winner = try await store.upsert(
            Project(
                analysis: ProjectAnalysis(name: "shop", path: "/c/shop"), slug: "shop", localDomain: "shop.orbit.test"))
        let late = Project(
            analysis: ProjectAnalysis(name: "shop", path: "/c/shop"), slug: "shop", localDomain: "shop.orbit.test")
        let error = try #require(await duplicateError(inserting: late, into: store))

        let resolved = try await registry.resolveRace(late, after: error)

        #expect(resolved.id == winner.id)
    }
}
