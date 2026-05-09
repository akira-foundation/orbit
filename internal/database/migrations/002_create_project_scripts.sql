-- +goose Up
CREATE TABLE project_scripts (
    id         TEXT    PRIMARY KEY,
    project_id TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    command    TEXT    NOT NULL,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_scripts_project ON project_scripts(project_id);

-- +goose Down
DROP INDEX IF EXISTS idx_scripts_project;
DROP TABLE project_scripts;
