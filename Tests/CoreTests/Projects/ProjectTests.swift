import Foundation
import Testing

@testable import Core

@Suite struct ProjectTests {
    @Test func analysisWithoutProcessesGivesASingleBackend() {
        let analysis = ProjectAnalysis(name: "app", path: "/tmp/app", runtimeKind: .phpFPM, phpVersion: "8.3")
        let project = Project(analysis: analysis, slug: "app", localDomain: "app.orbit.test")

        #expect(project.processes == [ProcessSpec(role: .backend, kind: .phpFPM, phpVersion: "8.3")])
    }

    @Test func analysisWithProcessesKeepsThem() {
        let processes = [
            ProcessSpec(role: .backend, kind: .phpFPM, phpVersion: "8.3"),
            ProcessSpec(role: .frontend, kind: .command, command: "npm run dev"),
        ]
        let analysis = ProjectAnalysis(name: "app", path: "/tmp/app", runtimeKind: .phpFPM, processes: processes)
        let project = Project(analysis: analysis, slug: "app", localDomain: "app.orbit.test")

        #expect(project.processes == processes)
    }

    @Test func initCopiesTheAnalysisAndSortsScripts() {
        let analysis = ProjectAnalysis(
            name: "app", path: "/tmp/app", packageManager: "bun", framework: "vite", devCommand: "bun run dev",
            devPort: 5173, nodeVersion: "22", scripts: ["test": "vitest", "build": "vite build"])
        let now = Date(timeIntervalSince1970: 1_000)
        let project = Project(
            analysis: analysis, slug: "app", localDomain: "app.orbit.test", parkedFolder: "/tmp", now: now)

        #expect(project.id == project.id.lowercased())
        #expect(UUID(uuidString: project.id) != nil)
        #expect(project.scripts.map(\.name) == ["build", "test"])
        #expect(project.packageManager == "bun")
        #expect(project.devPort == 5173)
        #expect(project.parkedFolder == "/tmp")
        #expect(project.status == .stopped)
        #expect(project.createdAt == now)
        #expect(project.updatedAt == now)
    }
}
