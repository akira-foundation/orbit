-- +goose Up
CREATE TABLE IF NOT EXISTS project_services (
    id          TEXT    PRIMARY KEY,
    project_id  TEXT    NOT NULL,
    engine      TEXT    NOT NULL,
    enabled     INTEGER NOT NULL DEFAULT 0,
    config_json TEXT    NOT NULL DEFAULT '{}',
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE (project_id, engine)
);

CREATE TABLE IF NOT EXISTS service_binaries (
    engine       TEXT NOT NULL,
    version      TEXT NOT NULL,
    path         TEXT NOT NULL,
    hash         TEXT NOT NULL,
    installed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (engine, version)
);

-- +goose Down
DROP TABLE service_binaries;
DROP TABLE project_services;
