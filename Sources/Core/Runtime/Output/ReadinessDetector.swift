public enum ReadinessDetector {
    public static let phrases = [
        "ready in",
        "ready - started",
        "local:",
        "localhost:",
        "listening on",
        "server running",
        "server started",
        "compiled successfully",
        "started development server",
        "uvicorn running on",
        "application startup complete",
        "starting development server at",
        "running on http",
    ]

    public static func looksReady(_ line: String) -> Bool {
        let lowered = ANSI.strip(line).lowercased()
        return phrases.contains { lowered.contains($0) }
    }
}
