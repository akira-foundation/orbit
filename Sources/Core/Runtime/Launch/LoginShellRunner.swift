import Dispatch
import Foundation
import Synchronization

public struct LoginShellRunner: LoginShellRunning {
    public init() {}

    public func run(shell: String, script: String, timeout: Duration) async -> String? {
        await withCheckedContinuation { continuation in
            let completion = Completion(continuation)
            let process = Process()
            process.executableURL = URL(fileURLWithPath: shell)
            process.arguments = ["-lic", script]
            process.standardInput = FileHandle.nullDevice
            process.standardError = FileHandle.nullDevice
            let pipe = Pipe()
            process.standardOutput = pipe

            let output = Mutex<Data>(Data())
            let drained = DispatchGroup()
            let reader = pipe.fileHandleForReading
            process.terminationHandler = { _ in
                _ = drained.wait(timeout: .now() + 0.5)
                completion.finish(String(decoding: output.withLock { $0 }, as: UTF8.self))
            }
            drained.enter()
            do {
                try process.run()
            } catch {
                drained.leave()
                completion.finish(nil)
                return
            }
            try? pipe.fileHandleForWriting.close()
            DispatchQueue.global().async {
                while case let chunk = reader.availableData, !chunk.isEmpty {
                    output.withLock { $0.append(chunk) }
                }
                drained.leave()
            }
            let seconds = Double(timeout.components.seconds) + Double(timeout.components.attoseconds) / 1e18
            DispatchQueue.global().asyncAfter(deadline: .now() + seconds) {
                if completion.finish(nil) {
                    process.terminate()
                }
            }
        }
    }
}

extension LoginShellRunner {
    private final class Completion: Sendable {
        private let continuation: Mutex<CheckedContinuation<String?, Never>?>

        init(_ continuation: CheckedContinuation<String?, Never>) {
            self.continuation = Mutex(continuation)
        }

        @discardableResult
        func finish(_ value: String?) -> Bool {
            let pending = continuation.withLock { state in
                defer { state = nil }
                return state
            }
            pending?.resume(returning: value)
            return pending != nil
        }
    }
}
