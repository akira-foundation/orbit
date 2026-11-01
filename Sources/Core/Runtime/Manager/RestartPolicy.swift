public struct RestartPolicy: Sendable {
    public static let maximumAttempts = 5
    public static let maximumDelay: Duration = .seconds(16)

    public static func delay(forAttempt attempt: Int) -> Duration? {
        guard (1...maximumAttempts).contains(attempt) else { return nil }
        return min(.seconds(1 << (attempt - 1)), maximumDelay)
    }
}
