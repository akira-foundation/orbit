import Foundation
import Synchronization

public actor EventLog {
    public static let bufferSize = 4096
    public static let batchSize = 200
    public static let flushInterval: Duration = .milliseconds(250)

    private let store: Store?
    private let clock: any RuntimeClock
    private let continuation: AsyncStream<EventRecord>.Continuation?
    private let enqueued = Atomic<Int>(0)
    private var received = 0
    private var pending: [EventRecord] = []
    private var finished = false
    private var drainWaiters: [(target: Int, continuation: CheckedContinuation<Void, Never>)] = []
    private var closeWaiters: [CheckedContinuation<Void, Never>] = []
    private var lastWrite: Task<Void, Never>?
    public private(set) var lastWriteError: (any Error)?

    public init(store: Store?, clock: any RuntimeClock) {
        self.store = store
        self.clock = clock
        guard store != nil else {
            continuation = nil
            finished = true
            return
        }
        let (stream, continuation) = AsyncStream.makeStream(
            of: EventRecord.self, bufferingPolicy: .bufferingOldest(Self.bufferSize))
        self.continuation = continuation
        Task { await self.run(stream) }
    }

    public nonisolated func push(_ record: EventRecord) {
        guard let continuation else { return }
        var record = record
        if record.timestamp == 0 {
            record.timestamp = Int64((clock.now.timeIntervalSince1970 * 1_000_000_000).rounded(.towardZero))
        }
        if case .enqueued = continuation.yield(record) {
            enqueued.add(1, ordering: .releasing)
        }
    }

    public func history(projectID: String, since: Int64, limit: Int) async throws -> [LogLine] {
        guard let store else { return [] }
        return try await store.events(projectID: projectID, since: since, limit: limit)
    }

    public func flush() async {
        let target = enqueued.load(ordering: .acquiring)
        if !finished && received < target {
            await withCheckedContinuation { drainWaiters.append((target, $0)) }
        }
        await writePending()
    }

    public func close() async {
        continuation?.finish()
        guard !finished else { return }
        await withCheckedContinuation { closeWaiters.append($0) }
    }

    private func run(_ stream: AsyncStream<EventRecord>) async {
        await withTaskGroup(of: Void.self) { group in
            group.addTask { [clock] in
                while (try? await clock.sleep(for: Self.flushInterval)) != nil {
                    await self.writePending()
                }
            }
            for await record in stream {
                pending.append(record)
                received += 1
                resumeDrainWaiters()
                if pending.count >= Self.batchSize {
                    await writePending()
                }
            }
            group.cancelAll()
        }
        await writePending()
        finished = true
        resumeDrainWaiters()
        closeWaiters.forEach { $0.resume() }
        closeWaiters.removeAll()
    }

    private func resumeDrainWaiters() {
        let ready = drainWaiters.filter { finished || $0.target <= received }
        drainWaiters.removeAll { finished || $0.target <= received }
        ready.forEach { $0.continuation.resume() }
    }

    private func writePending() async {
        if let store, !pending.isEmpty {
            let batch = pending
            pending.removeAll(keepingCapacity: true)
            let previous = lastWrite
            lastWrite = Task {
                await previous?.value
                do {
                    try await store.appendEvents(batch)
                } catch {
                    lastWriteError = error
                }
            }
        }
        await lastWrite?.value
    }
}
