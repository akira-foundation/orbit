import Foundation
import Synchronization

public struct LsofRunner: Sendable {
    static let executable = URL(fileURLWithPath: "/usr/sbin/lsof")
    static let environment = ["LC_ALL": "C", "LANG": "C", "PATH": "/usr/bin:/bin:/usr/sbin:/sbin"]

    public init() {}

    public func run(host: String, port: Int) throws -> String {
        let address = "\(host):\(port)"
        let process = Process()
        process.executableURL = Self.executable
        process.arguments = ["-nP", "-a", "-iTCP@\(address)", "-sTCP:LISTEN", "-Fp"]
        process.environment = Self.environment
        let output = Pipe()
        let errors = Pipe()
        process.standardInput = FileHandle.nullDevice
        process.standardOutput = output
        process.standardError = errors
        do {
            try process.run()
        } catch {
            throw RuntimeError.message("inspect listener \(address): \(error.localizedDescription)")
        }
        let errorData = Mutex(Data())
        let reading = DispatchGroup()
        let errorHandle = errors.fileHandleForReading
        DispatchQueue.global().async(group: reading) {
            let data = errorHandle.readDataToEndOfFile()
            errorData.withLock { $0 = data }
        }
        let standardOutput = String(decoding: output.fileHandleForReading.readDataToEndOfFile(), as: UTF8.self)
        reading.wait()
        process.waitUntilExit()
        let standardError = String(decoding: errorData.withLock { $0 }, as: UTF8.self)
        guard standardError.isEmpty else {
            throw RuntimeError.message(
                "inspect listener \(address): \(standardError.trimmingCharacters(in: .whitespacesAndNewlines))"
            )
        }
        let status = process.terminationStatus
        guard process.terminationReason == .exit, status == 0 else {
            if process.terminationReason == .exit, status == 1, standardOutput.isEmpty { return "" }
            throw RuntimeError.message("inspect listener \(address): lsof exited with status \(status)")
        }
        return standardOutput
    }
}
