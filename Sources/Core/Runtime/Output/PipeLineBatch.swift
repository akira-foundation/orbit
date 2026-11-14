public struct PipeLineBatch: Sendable, Equatable {
    public let lines: [String]
    public let overflowedLines: Int

    public init(lines: [String], overflowedLines: Int) {
        self.lines = lines
        self.overflowedLines = overflowedLines
    }
}
