public struct ParkedSyncResult: Sendable, Equatable {
    public struct Skip: Sendable, Equatable {
        public let path: String
        public let reason: String

        public init(path: String, reason: String) {
            self.path = path
            self.reason = reason
        }
    }

    public var added: [Project]
    public var removed: [Project]
    public var skipped: [Skip]

    public init(added: [Project] = [], removed: [Project] = [], skipped: [Skip] = []) {
        self.added = added
        self.removed = removed
        self.skipped = skipped
    }
}
