package services

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SetEnabled(ctx context.Context, projectID, engine string, enabled bool) error {
	v := 0
	if enabled {
		v = 1
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO project_services (id, project_id, engine, enabled)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (project_id, engine)
		DO UPDATE SET enabled = excluded.enabled,
		              updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		uuid.NewString(), projectID, engine, v)
	return err
}

func (s *Store) EnabledEngines(ctx context.Context, projectID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT engine FROM project_services WHERE project_id = ? AND enabled = 1 ORDER BY engine`,
		projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) ProjectsUsing(ctx context.Context, engine string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id FROM project_services WHERE engine = ? AND enabled = 1`,
		engine)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (s *Store) RecordBinary(ctx context.Context, engine, version, path, hash string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO service_binaries (engine, version, path, hash)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (engine, version)
		DO UPDATE SET path = excluded.path, hash = excluded.hash,
		              installed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		engine, version, path, hash)
	return err
}

func (s *Store) DeleteBinaries(ctx context.Context, engine string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM service_binaries WHERE engine = ?`, engine)
	return err
}

func (s *Store) Binary(ctx context.Context, engine, version string) (InstalledBinary, bool, error) {
	var b InstalledBinary
	err := s.db.QueryRowContext(ctx,
		`SELECT engine, version, path, hash, installed_at FROM service_binaries WHERE engine = ? AND version = ?`,
		engine, version).Scan(&b.Engine, &b.Version, &b.Path, &b.Hash, &b.InstalledAt)
	if err == sql.ErrNoRows {
		return InstalledBinary{}, false, nil
	}
	if err != nil {
		return InstalledBinary{}, false, err
	}
	return b, true, nil
}
