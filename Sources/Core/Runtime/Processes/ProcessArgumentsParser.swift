enum ProcessArgumentsParser {
    static func parse(_ buffer: [UInt8]) throws -> String {
        let count = MemoryLayout<Int32>.size
        guard buffer.count >= count else { throw malformed }
        let argc = buffer.withUnsafeBytes { $0.loadUnaligned(as: Int32.self) }
        guard argc >= 0 else { throw malformed }
        var index = try skipString(in: buffer, from: count)
        while index < buffer.count && buffer[index] == 0 {
            index += 1
        }
        var arguments: [String] = []
        for _ in 0..<argc {
            let end = try skipString(in: buffer, from: index)
            arguments.append(String(decoding: buffer[index..<(end - 1)], as: UTF8.self))
            index = end
        }
        return arguments.joined(separator: " ")
    }

    private static let malformed = RuntimeError.message("inspect process: malformed arguments")

    private static func skipString(in buffer: [UInt8], from start: Int) throws -> Int {
        guard start < buffer.count, let terminator = buffer[start...].firstIndex(of: 0) else { throw malformed }
        return terminator + 1
    }
}
