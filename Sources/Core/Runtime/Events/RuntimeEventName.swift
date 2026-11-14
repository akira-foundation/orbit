public enum RuntimeEventName: String, Sendable {
    case starting = "runtime:starting"
    case running = "runtime:running"
    case stopped = "runtime:stopped"
    case error = "runtime:error"
    case log = "runtime:log"
    case crash = "runtime:crash"
}
