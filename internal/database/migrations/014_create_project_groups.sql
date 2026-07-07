-- +goose Up
CREATE TABLE IF NOT EXISTS project_groups (
    id         TEXT    PRIMARY KEY,
    name       TEXT    NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS project_group_members (
    group_id   TEXT NOT NULL REFERENCES project_groups(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, project_id)
);

CREATE INDEX IF NOT EXISTS idx_group_members_group ON project_group_members(group_id);

-- +goose Down
DROP INDEX IF EXISTS idx_group_members_group;
DROP TABLE project_group_members;
DROP TABLE project_groups;
