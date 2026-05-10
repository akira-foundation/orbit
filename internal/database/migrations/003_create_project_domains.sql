-- +goose Up
CREATE TABLE IF NOT EXISTS project_domains (
    id          TEXT    PRIMARY KEY,
    project_id  TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    domain      TEXT    NOT NULL UNIQUE,
    target_port INTEGER NOT NULL DEFAULT 0,
    enabled     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_domains_project ON project_domains(project_id);
CREATE INDEX IF NOT EXISTS idx_domains_enabled ON project_domains(enabled);

-- +goose Down
DROP INDEX IF EXISTS idx_domains_enabled;
DROP INDEX IF EXISTS idx_domains_project;
DROP TABLE project_domains;
