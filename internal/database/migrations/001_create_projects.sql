-- +goose Up
CREATE TABLE IF NOT EXISTS projects (
    id                 TEXT    PRIMARY KEY,
    name               TEXT    NOT NULL,
    path               TEXT    NOT NULL UNIQUE,
    slug               TEXT    NOT NULL UNIQUE,
    local_domain       TEXT    NOT NULL,
    detected_framework TEXT    NOT NULL,
    package_manager    TEXT    NOT NULL,
    dev_command        TEXT    NOT NULL,
    dev_port           INTEGER NOT NULL DEFAULT 0,
    status             TEXT    NOT NULL DEFAULT 'stopped',
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- +goose Down
DROP TABLE projects;
