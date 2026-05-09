package database

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// Open connects to the SQLite database at path, runs migrations, and returns
// the connection. WAL mode and foreign-key enforcement are always enabled.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

// migration holds one versioned DDL step.
type migration struct {
	version int
	sql     string
}

// migrations is the ordered list of schema changes.
// Each entry is applied exactly once, tracked via schema_migrations.
var migrations = []migration{
	{1, `CREATE TABLE IF NOT EXISTS projects (
		id                 TEXT PRIMARY KEY,
		name               TEXT NOT NULL,
		path               TEXT NOT NULL UNIQUE,
		slug               TEXT NOT NULL UNIQUE,
		local_domain       TEXT NOT NULL,
		detected_framework TEXT NOT NULL,
		package_manager    TEXT NOT NULL,
		dev_command        TEXT NOT NULL,
		dev_port           INTEGER NOT NULL DEFAULT 0,
		status             TEXT NOT NULL DEFAULT 'stopped',
		created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`},
	{2, `CREATE TABLE IF NOT EXISTS project_scripts (
		id         TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		name       TEXT NOT NULL,
		command    TEXT NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	)`},
	{3, `CREATE INDEX IF NOT EXISTS idx_scripts_project ON project_scripts(project_id)`},
}

func migrate(db *sql.DB) error {
	// Version tracking table
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version    INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, m := range migrations {
		var count int
		err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, m.version).Scan(&count)
		if err != nil {
			return fmt.Errorf("check migration %d: %w", m.version, err)
		}
		if count > 0 {
			continue // already applied
		}
		if _, err := db.Exec(m.sql); err != nil {
			return fmt.Errorf("apply migration %d: %w", m.version, err)
		}
		if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, m.version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}
	return nil
}
