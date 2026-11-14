import Foundation
import Synchronization
import Testing

@testable import Core

struct RuntimeSessionTests {
    private func session(_ clock: TestClock = TestClock()) -> RuntimeSession {
        RuntimeSession(projectID: "p", sessionID: "0123456789abcdef", clock: clock)
    }

    @Test func stopCallbackInstallRaceFiresExactlyOnce() async {
        for _ in 0..<1000 {
            let session = session()
            let calls = Atomic<Int>(0)
            await withTaskGroup(of: Void.self) { group in
                group.addTask { session.installStopCallback { calls.add(1, ordering: .relaxed) } }
                group.addTask { session.requestShutdown() }
            }
            #expect(calls.load(ordering: .relaxed) == 1)
        }
    }

    @Test func stopRequestReleasesSignalAndCallbacksOnce() async {
        let session = session()
        let calls = Atomic<Int>(0)
        session.installStopCallback { calls.add(1, ordering: .relaxed) }

        session.requestStop()
        session.requestStop()
        session.requestShutdown()
        await session.stopSignal()

        #expect(calls.load(ordering: .relaxed) == 1)
        #expect(session.shutdownRequested)
    }

    @Test func shutdownRequestDoesNotReleaseTheStopSignal() async {
        let session = session()
        let released = Atomic<Bool>(false)
        let waiter = Task {
            await session.stopSignal()
            released.store(true, ordering: .relaxed)
        }
        session.requestShutdown()
        try? await Task.sleep(for: .milliseconds(100))
        #expect(released.load(ordering: .relaxed) == false)

        session.requestStop()
        await waiter.value
        #expect(released.load(ordering: .relaxed) == true)
    }

    @Test func stopSignalWaiterReturnsWhenCancelled() async throws {
        let session = session()
        let released = Atomic<Bool>(false)
        let waiter = Task {
            await session.stopSignal()
            released.store(true, ordering: .relaxed)
        }
        try await Task.sleep(for: .milliseconds(100))
        #expect(released.load(ordering: .relaxed) == false)

        waiter.cancel()
        let deadline = ContinuousClock.now + .seconds(20)
        while !released.load(ordering: .relaxed) && ContinuousClock.now < deadline {
            try await Task.sleep(for: .milliseconds(5))
        }
        #expect(released.load(ordering: .relaxed) == true)
        #expect(!session.shutdownRequested)
    }

    @Test func stopRequestReleasesEveryWaiter() async throws {
        let session = session()
        let released = Atomic<Int>(0)
        let waiters = (0..<3).map { _ in
            Task {
                await session.stopSignal()
                released.add(1, ordering: .relaxed)
            }
        }
        try await Task.sleep(for: .milliseconds(100))
        #expect(released.load(ordering: .relaxed) == 0)

        session.requestStop()
        for waiter in waiters { await waiter.value }
        #expect(released.load(ordering: .relaxed) == 3)
    }

    @Test func firedStopCallbackIsReleased() throws {
        let session = session()
        weak var observed: TemporaryDirectory?
        do {
            let captured = try TemporaryDirectory()
            observed = captured
            session.installStopCallback { _ = captured.path }
        }
        #expect(observed != nil)

        session.requestShutdown()
        #expect(observed == nil)
    }

    @Test func callbackInstalledAfterShutdownFiresImmediately() {
        let session = session()
        let calls = Atomic<Int>(0)
        session.requestShutdown()

        session.installStopCallback { calls.add(1, ordering: .relaxed) }
        #expect(calls.load(ordering: .relaxed) == 1)

        session.requestShutdown()
        #expect(calls.load(ordering: .relaxed) == 1)
    }

    @Test func ringKeepsTheNewest2000AndCountsAll() {
        let session = session()
        for index in 0...RuntimeSession.logCapacity {
            session.addLog(LogLine(timestamp: .now, stream: .stdout, text: "\(index)"))
        }
        let logs = session.logs()
        #expect(logs.count == RuntimeSession.logCapacity)
        #expect(logs.first?.text == "1")
        #expect(logs.last?.text == "2000")
        #expect(session.logTotal == 2001)
    }

    @Test func runningSetsReadyAtOnceAndWasRunning() {
        let clock = TestClock()
        let session = session(clock)
        #expect(session.status == .starting)
        #expect(!session.wasRunning)

        clock.advance(by: 2)
        session.setStatus(.running)
        clock.advance(by: 5)
        session.setStatus(.running)

        let snapshot = session.snapshot(conns: 0)
        #expect(snapshot.readyAt == clock.now.addingTimeInterval(-5))
        #expect(snapshot.startupMs == 2000)
        #expect(session.wasRunning)
    }

    @Test func mutatorsBumpLastActivityButSupervisorDoesNot() {
        let clock = TestClock()
        let session = session(clock)
        clock.advance(by: 1)
        session.setSupervisor(ProcessIdentity(pid: 41, pgid: 41, startID: "s", executable: "sh", arguments: "sh"))
        #expect(session.lastActivity == clock.now.addingTimeInterval(-1))

        session.assign(Endpoint(host: "127.0.0.1", port: 5173))
        #expect(session.lastActivity == clock.now)
        #expect(session.endpoint == Endpoint(host: "127.0.0.1", port: 5173))
        #expect(session.supervisor?.pid == 41)
    }

    @Test func errorsSetStatusAndMessage() {
        let session = session()
        session.setError("boom")
        #expect(session.status == .error)
        #expect(session.snapshot(conns: 0).error == "boom")

        session.setShutdownError(RuntimeError.groupNotEmpty)
        #expect(session.snapshot(conns: 0).error == "runtime: process group is not empty")
        #expect(session.shutdownError?.contains(.groupNotEmpty) == true)

        session.markStopped()
        #expect(session.status == .stopped)
    }

    @Test func snapshotUptimeOnlyWhileStartingOrRunning() {
        let clock = TestClock()
        let session = session(clock)
        clock.advance(by: 3)
        #expect(session.snapshot(conns: 0).uptimeMs == 3000)

        session.setStatus(.running)
        clock.advance(by: 1)
        #expect(session.snapshot(conns: 2).uptimeMs == 4000)
        #expect(session.snapshot(conns: 2).conns == 2)

        session.markStopped()
        #expect(session.snapshot(conns: 0).uptimeMs == 0)
    }

    @Test func snapshotEncodesTheGoKeys() throws {
        let clock = TestClock(Date(timeIntervalSince1970: 1_800_000_000.75))
        let session = session(clock)
        session.assign(Endpoint(host: "127.0.0.1", port: 8000))

        let data = try JSONEncoder().encode(session.snapshot(conns: 1))
        let object = try #require(try JSONSerialization.jsonObject(with: data) as? [String: Any])

        #expect(object["projectId"] as? String == "p")
        #expect(object["status"] as? String == "starting")
        #expect(object["startedAt"] as? String == "2027-01-15T08:00:00Z")
        #expect(object["lastActivity"] as? String == "2027-01-15T08:00:00Z")
        #expect(object["port"] as? Int == 8000)
        #expect(object["pid"] as? Int == 0)
        #expect(object["conns"] as? Int == 1)
        #expect(object["uptimeMs"] as? Int == 0)
        #expect(object["readyAt"] == nil)
        #expect(object["startupMs"] == nil)
        #expect(object["error"] == nil)
        #expect(object["phase"] == nil)
        #expect(object["attempts"] == nil)
    }

    @Test func snapshotOmitsZeroStartupAndEmptyError() throws {
        let session = session()
        session.setStatus(.running)
        session.setError("")

        let data = try JSONEncoder().encode(session.snapshot(conns: 0))
        let object = try #require(try JSONSerialization.jsonObject(with: data) as? [String: Any])

        #expect(object["readyAt"] as? String == "2027-01-15T08:00:00Z")
        #expect(object["startupMs"] == nil)
        #expect(object["error"] == nil)
    }

    @Test func snapshotDecodesAnExplicitNullReadyAt() throws {
        let json = """
            {"projectId":"p","status":"stopped","pid":0,"port":0,"startedAt":"2027-01-15T08:00:00Z","readyAt":null,
            "uptimeMs":0,"conns":0,"lastActivity":"2027-01-15T08:00:00Z"}
            """
        let snapshot = try JSONDecoder().decode(SessionSnapshot.self, from: Data(json.utf8))
        #expect(snapshot.readyAt == nil)
        #expect(snapshot.status == .stopped)
    }

    @Test func snapshotDecodesAMissingReadyAt() throws {
        let json = """
            {"projectId":"p","status":"starting","pid":0,"port":0,"startedAt":"2027-01-15T08:00:00Z",
            "uptimeMs":0,"conns":0,"lastActivity":"2027-01-15T08:00:00Z"}
            """
        let snapshot = try JSONDecoder().decode(SessionSnapshot.self, from: Data(json.utf8))
        #expect(snapshot.readyAt == nil)
    }

    @Test func logLineEncodesAnRFC3339NanoTimestamp() throws {
        let fractional = LogLine(timestamp: Date(timeIntervalSince1970: 1_800_000_000.25), stream: .stdout, text: "a")
        let whole = LogLine(timestamp: Date(timeIntervalSince1970: 1_800_000_000), stream: .stderr, text: "b")

        let object = try #require(
            try JSONSerialization.jsonObject(with: JSONEncoder().encode([fractional, whole])) as? [[String: Any]])

        #expect(object[0]["timestamp"] as? String == "2027-01-15T08:00:00.25Z")
        #expect(object[1]["timestamp"] as? String == "2027-01-15T08:00:00Z")
        #expect(object[0]["stream"] as? String == "stdout")
        let decoded = try JSONDecoder().decode([LogLine].self, from: JSONEncoder().encode([fractional, whole]))
        #expect(decoded == [fractional, whole])
    }

    @Test func logLineRejectsLettersInTheFraction() {
        let json = #"{"timestamp":"2027-01-15T08:00:00.5e3Z","stream":"stdout","text":"a"}"#
        #expect(throws: DecodingError.self) {
            try JSONDecoder().decode(LogLine.self, from: Data(json.utf8))
        }
    }

    @Test func snapshotRoundTripsThroughJSON() throws {
        let clock = TestClock()
        let session = session(clock)
        clock.advance(by: 2)
        session.setStatus(.running)
        let snapshot = session.snapshot(conns: 0)
        let decoded = try JSONDecoder().decode(SessionSnapshot.self, from: JSONEncoder().encode(snapshot))
        #expect(decoded == snapshot)
    }
}
