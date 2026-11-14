@testable import Core

extension Store {
    func eventCount(projectID: String) throws -> Int {
        let rows = try db.query("SELECT COUNT(*) AS n FROM events WHERE project_id = ?;", [.text(projectID)])
        return Int(rows.first?.int("n") ?? 0)
    }
}
