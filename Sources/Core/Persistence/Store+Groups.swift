import Foundation

extension Store {
    public func createGroup(named name: String) throws -> ProjectGroup {
        try db.transaction {
            let next =
                try db.query("SELECT COALESCE(MAX(sort_order), -1) + 1 AS next FROM project_groups;")
                .first?.int("next") ?? 0
            let group = ProjectGroup(
                id: UUID().uuidString.lowercased(), name: name, sortOrder: Int(next), createdAt: .now)
            try db.run(
                "INSERT INTO project_groups (id, name, sort_order, created_at) VALUES (?, ?, ?, ?);",
                [
                    .text(group.id), .text(group.name), .int(Int64(group.sortOrder)),
                    .double(group.createdAt.timeIntervalSince1970),
                ])
            return group
        }
    }

    @discardableResult
    public func update(groupID: String, _ change: @Sendable (inout ProjectGroup) -> Void) throws -> ProjectGroup {
        try db.transaction {
            guard var group = try loadGroups(where: "id = ?", [.text(groupID)]).first else {
                throw StoreError.groupNotFound(groupID)
            }
            change(&group)
            try db.run(
                "UPDATE project_groups SET name = ?, sort_order = ? WHERE id = ?;",
                [.text(group.name), .int(Int64(group.sortOrder)), .text(groupID)])
            group.id = groupID
            return group
        }
    }

    public func groups() throws -> [ProjectGroup] {
        try db.transaction { try loadGroups(where: nil, []) }
    }

    public func deleteGroup(id: String) throws {
        guard try db.run("DELETE FROM project_groups WHERE id = ?;", [.text(id)]) > 0 else {
            throw StoreError.groupNotFound(id)
        }
    }

    public func addProject(_ projectID: String, toGroup groupID: String) throws {
        try db.transaction {
            guard try exists("project_groups", id: groupID) else { throw StoreError.groupNotFound(groupID) }
            guard try exists("projects", id: projectID) else { throw StoreError.projectNotFound(projectID) }
            try db.run(
                "INSERT OR IGNORE INTO project_group_members (group_id, project_id) VALUES (?, ?);",
                [.text(groupID), .text(projectID)])
        }
    }

    public func removeProject(_ projectID: String, fromGroup groupID: String) throws {
        try db.transaction {
            guard try exists("project_groups", id: groupID) else { throw StoreError.groupNotFound(groupID) }
            try db.run(
                "DELETE FROM project_group_members WHERE group_id = ? AND project_id = ?;",
                [.text(groupID), .text(projectID)])
        }
    }

    private func exists(_ table: String, id: String) throws -> Bool {
        try !db.query("SELECT 1 AS found FROM \(table) WHERE id = ?;", [.text(id)]).isEmpty
    }

    private func loadGroups(where condition: String?, _ bindings: [SQLValue]) throws -> [ProjectGroup] {
        let filter = condition.map { " WHERE \($0)" } ?? ""
        let rows = try db.query(
            "SELECT * FROM project_groups\(filter) ORDER BY sort_order, created_at, id;", bindings)
        return try rows.map { row in
            let id = try row.requiredText("id", in: "project_groups")
            let members = try db.query(
                """
                SELECT m.project_id FROM project_group_members m
                JOIN projects p ON p.id = m.project_id
                WHERE m.group_id = ? ORDER BY p.name COLLATE NOCASE, p.id;
                """, [.text(id)]
            ).map { try $0.requiredText("project_id", in: "project_group_members") }
            return ProjectGroup(
                id: id,
                name: try row.requiredText("name", in: "project_groups"),
                sortOrder: Int(try row.requiredInt("sort_order", in: "project_groups")),
                createdAt: Date(timeIntervalSince1970: try row.requiredDouble("created_at", in: "project_groups")),
                projectIDs: members)
        }
    }
}
