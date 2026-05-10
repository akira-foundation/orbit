-- +goose Up
CREATE TABLE IF NOT EXISTS runtime_logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    session_id TEXT    NOT NULL,
    ts         INTEGER NOT NULL,
    stream     TEXT    NOT NULL,
    text       TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_runtime_logs_project_ts ON runtime_logs(project_id, ts);
CREATE INDEX IF NOT EXISTS idx_runtime_logs_session    ON runtime_logs(session_id);

-- +goose Down
DROP INDEX IF EXISTS idx_runtime_logs_session;
DROP INDEX IF EXISTS idx_runtime_logs_project_ts;
DROP TABLE runtime_logs;
