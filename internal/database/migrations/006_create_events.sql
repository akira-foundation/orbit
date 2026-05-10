-- +goose Up
ALTER TABLE runtime_logs RENAME TO events;
ALTER TABLE events ADD COLUMN level  TEXT NOT NULL DEFAULT 'info';
ALTER TABLE events ADD COLUMN source TEXT NOT NULL DEFAULT 'runtime';

DROP INDEX IF EXISTS idx_runtime_logs_project_ts;
DROP INDEX IF EXISTS idx_runtime_logs_session;

CREATE INDEX IF NOT EXISTS idx_events_project_ts ON events(project_id, ts);
CREATE INDEX IF NOT EXISTS idx_events_session    ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_events_source     ON events(source);
CREATE INDEX IF NOT EXISTS idx_events_level      ON events(level);

-- +goose Down
DROP INDEX IF EXISTS idx_events_level;
DROP INDEX IF EXISTS idx_events_source;
DROP INDEX IF EXISTS idx_events_session;
DROP INDEX IF EXISTS idx_events_project_ts;
ALTER TABLE events RENAME TO runtime_logs;
CREATE INDEX IF NOT EXISTS idx_runtime_logs_project_ts ON runtime_logs(project_id, ts);
CREATE INDEX IF NOT EXISTS idx_runtime_logs_session    ON runtime_logs(session_id);
