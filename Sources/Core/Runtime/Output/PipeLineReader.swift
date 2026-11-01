import Darwin
import Foundation
import Synchronization

public final class PipeLineReader: Sendable {
    public static let chunkSize = 64 * 1024

    public let batches: AsyncStream<[String]>
    private let output: FileHandle
    private let continuation: AsyncStream<[String]>.Continuation
    private let consumed = DispatchSemaphore(value: 0)
    private let abandoned = Atomic<Bool>(false)

    public init(_ output: FileHandle) {
        self.output = output
        (batches, continuation) = AsyncStream.makeStream(of: [String].self)
        continuation.onTermination = { [weak self] _ in
            self?.abandoned.store(true, ordering: .releasing)
            self?.consumed.signal()
        }
    }

    public func start() {
        Thread.detachNewThread { self.run() }
    }

    public func acknowledge() {
        consumed.signal()
    }

    private func run() {
        var scanner = LineScanner()
        var buffer = [UInt8](repeating: 0, count: Self.chunkSize)
        while !abandoned.load(ordering: .acquiring) {
            let count = buffer.withUnsafeMutableBytes { Darwin.read(output.fileDescriptor, $0.baseAddress, $0.count) }
            if count < 0 && errno == EINTR {
                continue
            }
            guard count > 0 else { break }
            deliver(scanner.feed(buffer[0..<count]))
        }
        deliver(scanner.finish())
        continuation.finish()
    }

    private func deliver(_ lines: [String]) {
        guard !lines.isEmpty, !abandoned.load(ordering: .acquiring) else { return }
        continuation.yield(lines)
        consumed.wait()
    }
}
