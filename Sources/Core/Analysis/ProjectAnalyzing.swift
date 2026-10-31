public protocol ProjectAnalyzing: Sendable {
    func analyze(path: String) throws -> ProjectAnalysis
}
