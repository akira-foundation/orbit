public struct KernelProcessSnapshot: Sendable, Equatable {
    public let pid: Int32
    public let pgid: Int32
    public let ppid: Int32
    public let startID: String
    public let command: String

    public init(pid: Int32, pgid: Int32, ppid: Int32, startID: String, command: String) {
        self.pid = pid
        self.pgid = pgid
        self.ppid = ppid
        self.startID = startID
        self.command = command
    }

    func isSameProcess(as other: KernelProcessSnapshot) -> Bool {
        pid == other.pid && pgid == other.pgid && startID == other.startID
    }
}
