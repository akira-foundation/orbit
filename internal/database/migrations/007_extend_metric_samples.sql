-- +goose Up
ALTER TABLE metric_samples ADD COLUMN req_count    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN err_count    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN p50_ms       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN p95_ms       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN p99_ms       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN bytes_in     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN bytes_out    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN http_reqs    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN ws_reqs      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN mem_kb       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN cpu_pct      REAL    NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN crashes      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN autostops    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metric_samples ADD COLUMN wake_ms      INTEGER NOT NULL DEFAULT 0;

-- +goose Down
-- SQLite ALTER cannot drop columns; recreate the table to roll back.
CREATE TABLE metric_samples_new (
    project_id TEXT    NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    ts         INTEGER NOT NULL,
    status     TEXT    NOT NULL,
    port       INTEGER NOT NULL DEFAULT 0,
    conns      INTEGER NOT NULL DEFAULT 0,
    uptime_ms  INTEGER NOT NULL DEFAULT 0,
    attempts   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (project_id, ts)
);
INSERT INTO metric_samples_new SELECT project_id, ts, status, port, conns, uptime_ms, attempts FROM metric_samples;
DROP TABLE metric_samples;
ALTER TABLE metric_samples_new RENAME TO metric_samples;
CREATE INDEX IF NOT EXISTS idx_metric_samples_ts ON metric_samples(ts);
