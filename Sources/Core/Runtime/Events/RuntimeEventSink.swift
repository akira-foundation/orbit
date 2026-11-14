public protocol RuntimeEventSink: Sendable {
    func emit(_ event: RuntimeEvent)
}
