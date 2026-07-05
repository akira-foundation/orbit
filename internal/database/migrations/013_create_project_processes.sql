-- +goose Up
CREATE TABLE IF NOT EXISTS project_processes (
    id          TEXT    PRIMARY KEY,
    project_id  TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role        TEXT    NOT NULL DEFAULT 'backend',
    kind        TEXT    NOT NULL DEFAULT 'command',
    work_dir    TEXT    NOT NULL DEFAULT '',
    command     TEXT    NOT NULL DEFAULT '',
    port        INTEGER NOT NULL DEFAULT 0,
    php_version TEXT    NOT NULL DEFAULT '',
    sort_order  INTEGER NOT NULL DEFAULT 0,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_processes_project ON project_processes(project_id);

INSERT INTO project_processes (id, project_id, role, kind, work_dir, command, port, php_version, sort_order, created_at)
SELECT lower(hex(randomblob(16))), id, 'backend', runtime_kind, '', dev_command, dev_port, php_version, 0, created_at
FROM projects;

-- +goose Down
DROP INDEX IF EXISTS idx_processes_project;
DROP TABLE project_processes;
