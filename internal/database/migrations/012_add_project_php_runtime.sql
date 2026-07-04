-- +goose Up
ALTER TABLE projects ADD COLUMN runtime_kind TEXT NOT NULL DEFAULT 'command';
ALTER TABLE projects ADD COLUMN php_version TEXT NOT NULL DEFAULT '';

-- +goose Down
CREATE TABLE projects_new (
    id                 TEXT    PRIMARY KEY,
    name               TEXT    NOT NULL,
    path               TEXT    NOT NULL,
    slug               TEXT    NOT NULL,
    local_domain       TEXT    NOT NULL,
    detected_framework TEXT    NOT NULL DEFAULT '',
    package_manager    TEXT    NOT NULL DEFAULT '',
    dev_command        TEXT    NOT NULL DEFAULT '',
    dev_port           INTEGER NOT NULL DEFAULT 0,
    status             TEXT    NOT NULL DEFAULT 'stopped',
    secure             INTEGER NOT NULL DEFAULT 0,
    installed_hash     TEXT    NOT NULL DEFAULT '',
    node_version       TEXT    NOT NULL DEFAULT '',
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);
INSERT INTO projects_new
SELECT id, name, path, slug, local_domain, detected_framework, package_manager,
       dev_command, dev_port, status, secure, installed_hash, node_version, created_at, updated_at FROM projects;
DROP TABLE projects;
ALTER TABLE projects_new RENAME TO projects;
