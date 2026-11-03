enum Poll {
    static func until(within limit: Duration = .seconds(20), _ condition: () async throws -> Bool) async -> Bool {
        let clock = ContinuousClock()
        let deadline = clock.now.advanced(by: limit)
        while clock.now < deadline {
            if (try? await condition()) == true {
                return true
            }
            try? await clock.sleep(for: .milliseconds(5))
        }
        return (try? await condition()) == true
    }
}
