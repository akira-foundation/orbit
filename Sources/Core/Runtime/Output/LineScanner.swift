public struct LineScanner: Sendable {
    public static let maximumLineLength = 1_048_576

    public private(set) var overflowed = false

    private var pending: [UInt8] = []

    public init() {}

    public mutating func feed(_ bytes: some Sequence<UInt8>) -> [String] {
        var lines: [String] = []
        for byte in bytes {
            if overflowed {
                break
            }
            if byte == 0x0A {
                lines.append(takeLine())
            } else {
                pending.append(byte)
                if pending.count >= Self.maximumLineLength {
                    overflowed = true
                    pending = []
                }
            }
        }
        return lines
    }

    public mutating func finish() -> [String] {
        guard !overflowed, !pending.isEmpty else {
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
