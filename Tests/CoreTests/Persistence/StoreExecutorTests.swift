import Darwin
import Foundation
import Testing

@testable import Core

struct StoreExecutorTests {
    @Test(.disabled(if: ThreadSanitizer.isLoaded, ThreadSanitizer.exitTestsUnsupported))
    func writesWaitingOnAnotherConnectionLeaveTheCooperativePoolFree() async {
        await #expect(processExitsWith: .success) {
            guard let directory = try? TemporaryDirectory() else { exit(EXIT_FAILURE) }
            let path = directory.path + "/orbit.sqlite"
            let writers = ProcessInfo.processInfo.activeProcessorCount * 2
            guard let stores = try? (0..<writers).map({ _ in try Store(path: path) }),
                let holder = try? SQLiteDatabase(path: path),
                (try? holder.execute("BEGIN IMMEDIATE;")) != nil
            else { exit(EXIT_FAILURE) }
            let started = ContinuousClock.now
            let writes = Task {
                await withTaskGroup(of: Bool.self) { group in
                    for (index, store) in stores.enumerated() {
                        group.addTask {
                            let project = Project(
                                id: UUID().uuidString, name: "app\(index)", path: "/tmp/app\(index)",
                                slug: "app\(index)", localDomain: "app\(index).orbit.test")
                            return (try? await store.upsert(project)) != nil
                        }
                    }
                    return await group.allSatisfy { $0 }
                }
            }
            try? await Task.sleep(for: .milliseconds(200))
            let released = ContinuousClock.now - started
            guard (try? holder.execute("COMMIT;")) != nil else { exit(EXIT_FAILURE) }
            let succeeded = await writes.value
            exit(succeeded && released < .seconds(2) ? EXIT_SUCCESS : EXIT_FAILURE)
        }
    }
}
