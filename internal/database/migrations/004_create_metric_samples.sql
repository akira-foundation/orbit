-- +goose Up
CREATE TABLE IF NOT EXISTS metric_samples (
    project_id TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    ts         INTEGER NOT NULL,
    status     TEXT    NOT NULL,
    port       INTEGER NOT NULL DEFAULT 0,
    conns      INTEGER NOT NULL DEFAULT 0,
    uptime_ms  INTEGER NOT NULL DEFAULT 0,
    attempts   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_id, ts)
);

CREATE INDEX IF NOT EXISTS idx_metric_samples_ts ON metric_samples(ts);

-- +goose Down
DROP INDEX IF EXISTS idx_metric_samples_ts;
DROP TABLE metric_samples;
