import Foundation

public struct ProjectRegistry: Sendable {
    private let store: Store
    private let analyzer: any ProjectAnalyzing
    private let environment: OrbitEnvironment

    public init(store: Store, analyzer: any ProjectAnalyzing, environment: OrbitEnvironment) {
        self.store = store
        self.analyzer = analyzer
        self.environment = environment
    }

    public func analyze(path: String) throws -> ProjectAnalysis {
        let trimmed = path.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { throw AnalysisError.emptyPath }
        return try analyzer.analyze(path: trimmed)
    }

    public func domain(forSlug slug: String) -> String {
        "\(slug).\(environment.domainSuffix)"
    }

    public func add(path: String, overwrite: Bool, parkedFolder: String? = nil) async throws -> Project {
        let analysis = try analyze(path: path)
        if let reason = analysis.ambiguousReason {
            throw AnalysisError.ambiguousLayout(path: analysis.path, reason: reason)
        }
        let slug = Slug.make(from: analysis.name)
        let project = Project(
            analysis: analysis, slug: slug, localDomain: domain(forSlug: slug), parkedFolder: parkedFolder)
        do {
            return try await register(project, overwrite: overwrite)
        } catch let error as StoreError {
            guard case .duplicate = error else { throw error }
            return try await resolveRace(project, after: error)
        }
    }

    private func register(_ project: Project, overwrite: Bool) async throws -> Project {
        let existing = try await store.projects()
        if let samePath = existing.first(where: { $0.path == project.path }) {
            return samePath
        }
        guard let owner = existing.first(where: { $0.slug == project.slug }) else {
            return try await store.upsert(project)
        }
        guard overwrite else {
            throw ProjectConflict(existingName: owner.name, slug: owner.slug)
        }
        return try await store.upsert(project, replacing: owner.id)
    }

    func resolveRace(_ project: Project, after error: StoreError) async throws -> Project {
        let existing = try await store.projects()
        if let samePath = existing.first(where: { $0.path == project.path }) {
            return samePath
        }
        if let owner = existing.first(where: { $0.slug == project.slug }) {
            throw ProjectConflict(existingName: owner.name, slug: owner.slug)
        }
        throw error
    }
}
