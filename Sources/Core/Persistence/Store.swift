import Foundation

public actor Store {
    nonisolated let db: SQLiteDatabase

    public init(path: String) throws {
        let db = try SQLiteDatabase(path: path)
        try Self.migrate(db)
        self.db = db
    }

    public static func inMemory() throws -> Store {
        try Store(path: ":memory:")
    }

    public static func databaseURL(in base: URL, environment: OrbitEnvironment) -> URL {
        base.appendingPathComponent(environment.dataDirectoryName, isDirectory: true)
            .appendingPathComponent("orbit.sqlite")
    }

    public static func defaultPath(environment: OrbitEnvironment) throws -> String {
        let base = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
        let url = databaseURL(in: base, environment: environment)
        try FileManager.default.createDirectory(
            at: url.deletingLastPathComponent(), withIntermediateDirectories: true)
        return url.path
    }

    public func schemaVersion() throws -> Int32 {
        try db.userVersion
    }

    private static func migrate(_ db: SQLiteDatabase) throws {
        let migrations = StoreMigrations.all
        for index in 0..<migrations.count {
            try db.transaction {
                guard Int(try db.userVersion) <= index else { return }
                try db.execute(migrations[index])
                try db.execute("PRAGMA user_version = \(index + 1);")
            }
        }
        let version = try db.userVersion
        guard Int(version) <= migrations.count else {
            throw StoreError.unsupportedSchemaVersion(version)
        }
    }
}
