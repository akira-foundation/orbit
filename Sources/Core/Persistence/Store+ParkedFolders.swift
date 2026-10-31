import Foundation

extension Store {
    public func parkedFolders() throws -> [String] {
        try db.read {
            try db.query("SELECT path FROM parked_folders ORDER BY created_at, path;")
                .map { try $0.requiredText("path", in: "parked_folders") }
        }
    }

    public func addParkedFolder(_ path: String) throws {
        try db.run(
            "INSERT OR IGNORE INTO parked_folders (path, created_at) VALUES (?, ?);",
            [.text(CanonicalPath.of(path)), .double(Date.now.timeIntervalSince1970)])
    }

    public func deleteParkedProject(id: String, parkedFolder: String) throws -> Bool {
        try db.transaction {
            try db.run(
                "DELETE FROM projects WHERE id = ? AND parked_folder = ?;", [.text(id), .text(parkedFolder)]) > 0
        }
    }

    public func removeParkedFolder(_ path: String) throws {
        let canonical = CanonicalPath.of(path)
        try db.transaction {
            try db.run("UPDATE projects SET parked_folder = NULL WHERE parked_folder = ?;", [.text(canonical)])
            try db.run("DELETE FROM parked_folders WHERE path = ?;", [.text(canonical)])
        }
    }
}
