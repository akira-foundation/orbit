public enum LogStream: String, Sendable, Codable {
    case stdout
    case stderr
    case system
}
