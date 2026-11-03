enum ListenerOutputParser {
    private static let tolerated: Set<Character> = ["f", "t", "P", "n"]

    static func processIDs(_ output: String) throws -> [Int32] {
        guard !output.isEmpty else { return [] }
        let fields = output.hasSuffix("\n") ? String(output.dropLast()) : output
        guard !fields.isEmpty else { throw RuntimeError.message("ambiguous empty lsof field") }
        var unique: Set<Int32> = []
        for line in fields.split(separator: "\n", omittingEmptySubsequences: false) {
            guard line.count >= 2, let prefix = line.first else {
                throw RuntimeError.message("ambiguous lsof field \"\(line)\"")
            }
            guard prefix == "p" else {
                guard tolerated.contains(prefix) else { throw RuntimeError.message("ambiguous lsof field \"\(line)\"") }
                continue
            }
            guard let pid = Int32(line.dropFirst()), pid > 0 else {
                throw RuntimeError.message("invalid lsof PID field \"\(line)\"")
            }
            unique.insert(pid)
        }
        return unique.sorted()
    }
}
