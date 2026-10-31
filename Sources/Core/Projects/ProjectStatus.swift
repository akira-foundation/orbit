public enum ProjectStatus: String, Sendable, Codable {
    case stopped
    case starting
    case running
    case idle
    case suspended
    case error
}
