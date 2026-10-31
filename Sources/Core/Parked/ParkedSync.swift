import Foundation

public struct ParkedSync: Sendable {
    private let store: Store
    private let registry: ProjectRegistry

    public init(store: Store, registry: ProjectRegistry) {
        self.store = store
        self.registry = registry
    }

    public func run() async throws -> ParkedSyncResult {
        var result = ParkedSyncResult()
        try await removeMissing(into: &result)
        try await addDiscovered(into: &result)
        return result
    }

    private func removeMissing(into result: inout ParkedSyncResult) async throws {
        for project in try await store.projects() {
            guard let root = project.parkedFolder, Self.isGone(project.path, under: root) else { continue }
            try await store.deleteProject(id: project.id)
            result.removed.append(project)
        }
    }

    private static func isGone(_ path: String, under root: String) -> Bool {
        guard (try? FileManager.default.contentsOfDirectory(atPath: root)) != nil else { return false }
        var info = stat()
        return stat(path, &info) != 0 && errno == ENOENT
    }

    private func addDiscovered(into result: inout ParkedSyncResult) async throws {
        var known = Set(try await store.projects().map(\.id))
        var knownPaths = Set(try await store.projects().map(\.path))
        for folder in try await store.parkedFolders() {
            for path in ParkedScanner.discoverProjects(in: folder) where !knownPaths.contains(path) {
                do {
                    let project = try await registry.add(path: path, overwrite: false, parkedFolder: folder)
                    knownPaths.insert(path)
                    if known.insert(project.id).inserted {
                        result.added.append(project)
                    }
                } catch {
                    result.skipped.append(.init(path: path, reason: String(describing: error)))
                }
            }
        }
    }
}
