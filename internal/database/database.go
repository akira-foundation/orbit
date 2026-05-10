package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"orbit-app/internal/database/db"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// Open connects to the SQLite database at path, applies any pending goose
// migrations, and returns both the raw *sql.DB and a ready *db.Queries handle.
func Open(ctx context.Context, path string) (*sql.DB, *db.Queries, error) {
	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)&_pragma=busy_timeout(5000)",
		path,
	)
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, nil, err
	}
	if err := runMigrations(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return sqlDB, db.New(sqlDB), nil
}

func runMigrations(ctx context.Context, sqlDB *sql.DB) error {
	migFS, err := fs.Sub(embedMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("embed sub: %w", err)
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, sqlDB, migFS)
	if err != nil {
		return fmt.Errorf("goose provider: %w", err)
	}
	_, err = provider.Up(ctx)
	return err
}
