import Foundation

extension Store {
    @discardableResult
    public func upsert(_ project: Project, replacing replacedID: String? = nil) throws -> Project {
        try db.transaction {
            if let replacedID, replacedID != project.id {
                try db.run("DELETE FROM projects WHERE id = ?;", [.text(replacedID)])
            }
            try write(project)
        }
        return project
    }

    @discardableResult
    public func update(projectID: String, _ change: @Sendable (inout Project) -> Void) throws -> Project {
        try db.transaction {
            guard var project = try loadProjects(where: "id = ?", [.text(projectID)]).first else {
                throw StoreError.projectNotFound(projectID)
            }
            change(&project)
            project.id = projectID
            project.updatedAt = .now
            try write(project)
            return project
        }
    }

    public func project(id: String) throws -> Project? {
        try loadProjects(where: "id = ?", [.text(id)]).first
    }

    public func project(path: String) throws -> Project? {
        try loadProjects(where: "path = ?", [.text(path)]).first
    }

    public func project(domain: String) throws -> Project? {
        try loadProjects(where: "local_domain = ?", [.text(domain)]).first
    }

    public func projects() throws -> [Project] {
        try loadProjects(where: nil, [])
    }

    public func deleteProject(id: String) throws {
        guard try db.run("DELETE FROM projects WHERE id = ?;", [.text(id)]) > 0 else {
            throw StoreError.projectNotFound(id)
        }
    }

    private func write(_ project: Project) throws {
        try db.run(
            """
            INSERT INTO projects (
                id, name, path, slug, local_domain, framework, package_manager, dev_command, dev_port,
                node_version, runtime_kind, php_version, python_version, status, secure, pinned,
                installed_hash, parked_folder, created_at, updated_at
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                name = excluded.name, path = excluded.path, slug = excluded.slug,
                local_domain = excluded.local_domain, framework = excluded.framework,
                package_manager = excluded.package_manager, dev_command = excluded.dev_command,
                dev_port = excluded.dev_port, node_version = excluded.node_version,
                runtime_kind = excluded.runtime_kind, php_version = excluded.php_version,
                python_version = excluded.python_version, status = excluded.status, secure = excluded.secure,
                pinned = excluded.pinned, installed_hash = excluded.installed_hash,
                parked_folder = excluded.parked_folder, created_at = excluded.created_at,
                updated_at = excluded.updated_at;
            """,
            [
                .text(project.id), .text(project.name), .text(project.path), .text(project.slug),
                .text(project.localDomain), .optional(project.framework), .optional(project.packageManager),
                .optional(project.devCommand), .optional(project.devPort), .optional(project.nodeVersion),
                .text(project.runtimeKind.rawValue), .optional(project.phpVersion),
                .optional(project.pythonVersion), .text(project.status.rawValue), .bool(project.secure),
                .bool(project.pinned), .optional(project.installedHash), .optional(project.parkedFolder),
                .double(project.createdAt.timeIntervalSince1970),
                .double(project.updatedAt.timeIntervalSince1970),
            ])
        try db.run("DELETE FROM project_scripts WHERE project_id = ?;", [.text(project.id)])
        for script in project.scripts {
            try db.run(
                "INSERT INTO project_scripts (project_id, name, command) VALUES (?, ?, ?);",
                [.text(project.id), .text(script.name), .text(script.command)])
        }
        try db.run("DELETE FROM project_processes WHERE project_id = ?;", [.text(project.id)])
        for (index, process) in project.processes.enumerated() {
            try db.run(
                """
                INSERT INTO project_processes (
                    project_id, sort_order, role, kind, work_dir, command, port, php_version
                ) VALUES (?, ?, ?, ?, ?, ?, ?, ?);
                """,
                [
                    .text(project.id), .int(Int64(index)), .text(process.role.rawValue),
                    .text(process.kind.rawValue), .optional(process.workDirectory), .optional(process.command),
                    .optional(process.port), .optional(process.phpVersion),
                ])
        }
    }

    private func loadProjects(where condition: String?, _ bindings: [SQLValue]) throws -> [Project] {
        let filter = condition.map { " WHERE \($0)" } ?? ""
        let rows = try db.query("SELECT * FROM projects\(filter) ORDER BY name COLLATE NOCASE, id;", bindings)
        return try rows.map { row in
            let id = row.string("id") ?? ""
            let scripts = try db.query(
                "SELECT name, command FROM project_scripts WHERE project_id = ? ORDER BY name;", [.text(id)]
            ).map { ProjectScript(name: $0.string("name") ?? "", command: $0.string("command") ?? "") }
            let processes = try db.query(
                "SELECT * FROM project_processes WHERE project_id = ? ORDER BY sort_order;", [.text(id)]
            ).map { process in
                ProcessSpec(
                    role: process.string("role").flatMap(ProcessRole.init(rawValue:)) ?? .backend,
                    kind: process.string("kind").flatMap(RuntimeKind.init(rawValue:)) ?? .command,
                    workDirectory: process.string("work_dir"),
                    command: process.string("command"),
                    port: process.int("port").map(Int.init),
                    phpVersion: process.string("php_version"))
            }
            return Project(
                id: id,
                name: row.string("name") ?? "",
                path: row.string("path") ?? "",
                slug: row.string("slug") ?? "",
                localDomain: row.string("local_domain") ?? "",
                framework: row.string("framework"),
                packageManager: row.string("package_manager"),
                devCommand: row.string("dev_command"),
                devPort: row.int("dev_port").map(Int.init),
                runtimeKind: row.string("runtime_kind").flatMap(RuntimeKind.init(rawValue:)) ?? .command,
                nodeVersion: row.string("node_version"),
                phpVersion: row.string("php_version"),
                pythonVersion: row.string("python_version"),
                status: row.string("status").flatMap(ProjectStatus.init(rawValue:)) ?? .stopped,
                secure: row.bool("secure"),
                pinned: row.bool("pinned"),
                installedHash: row.string("installed_hash"),
                parkedFolder: row.string("parked_folder"),
                createdAt: Date(timeIntervalSince1970: row.double("created_at") ?? 0),
                updatedAt: Date(timeIntervalSince1970: row.double("updated_at") ?? 0),
                scripts: scripts,
                processes: processes)
        }
    }
}
