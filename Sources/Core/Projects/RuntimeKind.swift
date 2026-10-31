public enum RuntimeKind: String, Sendable, Codable {
    case command
    case phpFPM = "php-fpm"
    case python
}
