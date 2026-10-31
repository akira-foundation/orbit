import Foundation

extension Store {
    @discardableResult
    public func upsert(_ project: Project, replacing replacedID: String? = nil) throws -> Project {
        try reportingDuplicates {
            try db.transaction {
                if let replacedID, replacedID != project.id {
                    try db.run("DELETE FROM projects WHERE id = ?;", [.text(replacedID)])
                }
                try write(project)
            }
        }
        return project
    }

    @discardableResult
    public func update(projectID: String, _ change: @Sendable (inout Project) -> Void) throws -> Project {
        try reportingDuplicates {
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
    }

    private func reportingDuplicates<T>(_ body: () throws -> T) throws -> T {
        do {
            return try body()
        } catch let error as SQLiteError {
            guard let column = error.duplicateColumn else { throw error }
            throw StoreError.duplicate(column: column)
        }
    }

    public func project(id: String) throws -> Project? {
        try db.read { try loadProjects(where: "id = ?", [.text(id)]).first }
    }

    public func project(path: String) throws -> Project? {
        try db.read { try loadProjects(where: "path = ?", [.text(path)]).first }
    }

    public func project(domain: String) throws -> Project? {
        try db.read { try loadProjects(where: "local_domain = ?", [.text(domain)]).first }
    }

    public func projects() throws -> [Project] {
        try db.read { try loadProjects(where: nil, []) }
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
            let id = try row.requiredText("id", in: "projects")
            let scripts = try db.query(
                "SELECT name, command FROM project_scripts WHERE project_id = ? ORDER BY name;", [.text(id)]
            ).map {
                ProjectScript(
                    name: try $0.requiredText("name", in: "project_scripts"),
                    command: try $0.requiredText("command", in: "project_scripts"))
            }
            let processes = try db.query(
                "SELECT * FROM project_processes WHERE project_id = ? ORDER BY sort_order;", [.text(id)]
            ).map { process in
                ProcessSpec(
                    role: try process.requiredRaw("role", in: "project_processes"),
                    kind: try process.requiredRaw("kind", in: "project_processes"),
                    workDirectory: process.string("work_dir"),
                    command: process.string("command"),
                    port: process.int("port").map(Int.init),
                    phpVersion: process.string("php_version"))
            }
            return Project(
                id: id,
                name: try row.requiredText("name", in: "projects"),
                path: try row.requiredText("path", in: "projects"),
                slug: try row.requiredText("slug", in: "projects"),
                localDomain: try row.requiredText("local_domain", in: "projects"),
                framework: row.string("framework"),
                packageManager: row.string("package_manager"),
                devCommand: row.string("dev_command"),
                devPort: row.int("dev_port").map(Int.init),
                runtimeKind: try row.requiredRaw("runtime_kind", in: "projects"),
                nodeVersion: row.string("node_version"),
                phpVersion: row.string("php_version"),
                pythonVersion: row.string("python_version"),
                status: try row.requiredRaw("status", in: "projects"),
                secure: row.bool("secure"),
                pinned: row.bool("pinned"),
                installedHash: row.string("installed_hash"),
                parkedFolder: row.string("parked_folder"),
                createdAt: Date(timeIntervalSince1970: try row.requiredDouble("created_at", in: "projects")),
                updatedAt: Date(timeIntervalSince1970: try row.requiredDouble("updated_at", in: "projects")),
                scripts: scripts,
                processes: processes)
        }
    }
}
