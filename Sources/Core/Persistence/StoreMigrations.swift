enum StoreMigrations {
    static let all: [String] = [
        """
        CREATE TABLE projects (
            id TEXT PRIMARY KEY, name TEXT NOT NULL, path TEXT NOT NULL UNIQUE, slug TEXT NOT NULL UNIQUE,
            local_domain TEXT NOT NULL UNIQUE, framework TEXT, package_manager TEXT, dev_command TEXT, dev_port INTEGER,
            node_version TEXT, runtime_kind TEXT NOT NULL, php_version TEXT, python_version TEXT,
            status TEXT NOT NULL, secure INTEGER NOT NULL DEFAULT 0, pinned INTEGER NOT NULL DEFAULT 0,
            installed_hash TEXT, parked_folder TEXT, created_at REAL NOT NULL, updated_at REAL NOT NULL
        );
        CREATE TABLE project_scripts (
            project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
            name TEXT NOT NULL, command TEXT NOT NULL, PRIMARY KEY (project_id, name)
        );
        CREATE TABLE project_processes (
            project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE, sort_order INTEGER NOT NULL,
            role TEXT NOT NULL, kind TEXT NOT NULL, work_dir TEXT, command TEXT, port INTEGER, php_version TEXT,
            PRIMARY KEY (project_id, sort_order)
        );
        CREATE TABLE project_groups (
            id TEXT PRIMARY KEY, name TEXT NOT NULL, sort_order INTEGER NOT NULL, created_at REAL NOT NULL
        );
        CREATE TABLE project_group_members (
            group_id TEXT NOT NULL REFERENCES project_groups(id) ON DELETE CASCADE,
            project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
            PRIMARY KEY (group_id, project_id)
        );
        CREATE TABLE parked_folders (path TEXT PRIMARY KEY, created_at REAL NOT NULL);
        CREATE INDEX idx_projects_parked_folder ON projects(parked_folder);
        """
    ]
}
