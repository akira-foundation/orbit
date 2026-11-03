import Darwin
import Foundation
import Synchronization
import Testing

@testable import Core

struct PipeLineReaderTests {
    @Test func deliversEveryLineAndFinishesAtEndOfFile() async throws {
        let pipe = Pipe()
        let reader = PipeLineReader(pipe.fileHandleForReading)
        reader.start()

        try pipe.fileHandleForWriting.write(contentsOf: Data("one\ntwo\r\nthree".utf8))
        try pipe.fileHandleForWriting.close()
        var lines: [String] = []
        for await batch in reader.batches {
            lines += batch.lines
            reader.acknowledge()
        }

        #expect(lines == ["one", "two", "three"])
    }

    @Test func writerBlocksWhileABatchIsUnacknowledged() async throws {
        let pipe = Pipe()
        let reader = PipeLineReader(pipe.fileHandleForReading)
        reader.start()
        let writer = pipe.fileHandleForWriting.fileDescriptor
        _ = fcntl(writer, F_SETFL, fcntl(writer, F_GETFL) | O_NONBLOCK)
        let chunk = [UInt8](repeating: 0x0A, count: 4096)
        let limit = 64 * 1024 * 1024
        let bound = 1024 * 1024

        var written = 0
        var settledReads = 0
        while written < limit && settledReads < 200 {
            let count = chunk.withUnsafeBytes { Darwin.write(writer, $0.baseAddress, $0.count) }
            if count > 0 {
                written += count
                settledReads = 0
            } else {
                settledReads += 1
                try await Task.sleep(for: .milliseconds(1))
            }
        }

        #expect(written < bound)
        try pipe.fileHandleForWriting.close()
        for await _ in reader.batches {
            reader.acknowledge()
        }
    }

    @Test func keepsDeliveringAfterAnOverlongLine() async throws {
        let pipe = Pipe()
        let reader = PipeLineReader(pipe.fileHandleForReading)
        reader.start()
        let overlong = Data(repeating: 0x61, count: LineScanner.maximumLineLength + 4096)
        let written = Atomic<Bool>(false)

        Thread.detachNewThread {
            let output = overlong + Data("\nafter\n".utf8) + overlong
            guard (try? pipe.fileHandleForWriting.write(contentsOf: output)) != nil else {
                return
            }
            written.store(true, ordering: .relaxed)
            try? pipe.fileHandleForWriting.close()
        }
        var lines: [String] = []
        var overflowedLines = 0
        for await batch in reader.batches {
            lines += batch.lines
            overflowedLines += batch.overflowedLines
            reader.acknowledge()
        }

        #expect(await Poll.until(within: .seconds(10)) { written.load(ordering: .relaxed) })
        #expect(lines == ["after"])
        #expect(overflowedLines == 2)
    }
}
