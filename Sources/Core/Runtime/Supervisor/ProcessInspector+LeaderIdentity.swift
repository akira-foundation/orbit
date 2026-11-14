extension ProcessInspector {
    func leaderIdentity(of pid: Int32, within limit: Duration = .seconds(1)) async throws -> ProcessIdentity {
        let minimumReads = 10
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        var reads = 0
        while true {
            reads += 1
            let settled = reads >= minimumReads && clock.now >= deadline
            let observed: ProcessIdentity?
            do {
                observed = try self.identity(of: pid)
            } catch let failure where failure.contains(.identityMismatch("")) && !settled {
                observed = nil
            }
            if let observed, observed.pgid == pid || settled {
                return observed
            }
            do {
                try await clock.sleep(for: .milliseconds(1))
            } catch {
                throw RuntimeError.cancelled
            }
        }
    }
}
