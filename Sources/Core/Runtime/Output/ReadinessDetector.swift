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
        let lowered = ANSI.strip(line).utf8.map { (0x41...0x5A).contains($0) ? $0 + 0x20 : $0 }
        return phrases.contains { lowered.firstRange(of: $0.utf8) != nil }
    }
}
