public struct ProcessIdentity: Sendable, Hashable {
    public let pid: Int32
    public let pgid: Int32
    public let startID: String
    public let executable: String
    public let arguments: String

    public init(pid: Int32, pgid: Int32, startID: String, executable: String, arguments: String) {
        self.pid = pid
        self.pgid = pgid
        self.startID = startID
        self.executable = executable
        self.arguments = arguments
    }

    func isSameProcess(as other: ProcessIdentity) -> Bool {
        pid == other.pid && pgid == other.pgid && startID == other.startID
    }
}
