public struct LineScanner: Sendable {
    public static let maximumLineLength = 1_048_576

    public private(set) var overflowedLines = 0

    private var pending: [UInt8] = []
    private var discarding = false

    public init() {}

    public mutating func feed(_ bytes: some Sequence<UInt8>) -> [String] {
        var lines: [String] = []
        for byte in bytes {
            if byte == 0x0A {
                if !discarding {
                    lines.append(takeLine())
                }
                discarding = false
                continue
            }
            if discarding {
                continue
            }
            pending.append(byte)
            if pending.count >= Self.maximumLineLength {
                overflowedLines += 1
                discarding = true
                pending = []
            }
        }
        return lines
    }

    public mutating func finish() -> [String] {
        guard !pending.isEmpty else {
            return []
        }
        return [takeLine()]
    }

    private mutating func takeLine() -> String {
        if pending.last == 0x0D {
            pending.removeLast()
        }
        let line = String(decoding: pending, as: UTF8.self)
        pending.removeAll(keepingCapacity: true)
        return line
    }
}
