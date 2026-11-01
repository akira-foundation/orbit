public enum RuntimeEvent: Sendable, Equatable {
    case status(RuntimeEventName, SessionSnapshot)
    case log(projectID: String, LogLine)
}
