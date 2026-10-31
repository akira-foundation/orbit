import Foundation

extension Store {
    public func parkedFolders() throws -> [String] {
        try db.query("SELECT path FROM parked_folders ORDER BY created_at, path;")
            .map { try $0.requiredText("path", in: "parked_folders") }
    }

    public func addParkedFolder(_ path: String) throws {
        try db.run(
            "INSERT OR IGNORE INTO parked_folders (path, created_at) VALUES (?, ?);",
            [.text(path), .double(Date.now.timeIntervalSince1970)])
    }

    public func removeParkedFolder(_ path: String) throws {
        try db.transaction {
            try db.run("UPDATE projects SET parked_folder = NULL WHERE parked_folder = ?;", [.text(path)])
            try db.run("DELETE FROM parked_folders WHERE path = ?;", [.text(path)])
        }
    }
}
