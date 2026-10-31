import Foundation

@testable import Core

struct FixedAnalyzer: ProjectAnalyzing {
    func analyze(path: String) throws -> ProjectAnalysis {
        ProjectAnalysis(
            name: URL(fileURLWithPath: path).lastPathComponent, path: path, framework: "nestjs")
    }
}
