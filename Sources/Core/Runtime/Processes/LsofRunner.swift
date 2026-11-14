import Darwin
import Foundation

public struct LsofRunner: Sendable {
    static let environment = ["LC_ALL": "C", "LANG": "C", "PATH": "/usr/bin:/bin:/usr/sbin:/sbin"]

    private let executable: URL

    public init() {
        self.init(executable: URL(fileURLWithPath: "/usr/sbin/lsof"))
    }

    init(executable: URL) {
        self.executable = executable
    }

    public func run(host: String, port: Int) throws -> String {
        let address = "\(host):\(port)"
        let process: SpawnedProcess
        do {
            process = try PosixSpawner.spawn(
                SpawnRequest(
                    executable: executable.path(percentEncoded: false),
                    arguments: ["-nP", "-a", "-iTCP@\(address)", "-sTCP:LISTEN", "-Fp"],
                    environment: EnvironmentList(Self.environment.map { "\($0.key)=\($0.value)" }),
                    group: .inherit, captureOutput: true))
        } catch {
            throw RuntimeError.message("inspect listener \(address): \(error)")
        }
        guard let output = process.stdout, let errors = process.stderr else {
            throw RuntimeError.message("inspect listener \(address): lsof output was not captured")
        }
        let captured = PipeDrain.readToEnd([output.fileDescriptor, errors.fileDescriptor])
        let status = process.waitForExit()
        let standardOutput = String(decoding: captured[0], as: UTF8.self)
        let standardError = String(decoding: captured[1], as: UTF8.self)
        guard standardError.isEmpty else {
            throw RuntimeError.message(
                "inspect listener \(address): \(standardError.trimmingCharacters(in: .whitespacesAndNewlines))"
            )
        }
        switch status {
        case .exited(0):
            return standardOutput
        case .exited(1) where standardOutput.isEmpty:
            return ""
        case .exited(let code):
            throw RuntimeError.message("inspect listener \(address): lsof exited with status \(code)")
        case .signaled, .unknown:
            throw RuntimeError.message(
                "inspect listener \(address): \(status.failure("lsof").map(\.description) ?? "")")
        }
    }
}
