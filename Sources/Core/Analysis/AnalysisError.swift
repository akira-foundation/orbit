public enum AnalysisError: Error, Equatable, Sendable {
    case emptyPath
    case notADirectory(String)
    case noSupportedProject(String)
    case invalidManifest(path: String, reason: String)
}
