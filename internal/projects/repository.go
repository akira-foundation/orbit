package projects

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("project not found")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, p *Project) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = StatusStopped
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO projects
		(id, name, path, slug, local_domain, detected_framework, package_manager, dev_command, dev_port, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.Path, p.Slug, p.LocalDomain, p.DetectedFramework,
		p.PackageManager, p.DevCommand, p.DevPort, string(p.Status), p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return err
	}

	for i := range p.Scripts {
		s := &p.Scripts[i]
		if s.ID == "" {
			s.ID = uuid.NewString()
		}
		s.ProjectID = p.ID
		s.CreatedAt = now
		_, err := tx.ExecContext(ctx, `INSERT INTO project_scripts (id, project_id, name, command, created_at) VALUES (?, ?, ?, ?, ?)`,
			s.ID, s.ProjectID, s.Name, s.Command, s.CreatedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) List(ctx context.Context) ([]Project, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, path, slug, local_domain, detected_framework, package_manager, dev_command, dev_port, status, created_at, updated_at FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Project
	for rows.Next() {
		var p Project
		var status string
		if err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.Slug, &p.LocalDomain, &p.DetectedFramework,
			&p.PackageManager, &p.DevCommand, &p.DevPort, &status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Status = Status(status)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id string) (*Project, error) {
	var p Project
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT id, name, path, slug, local_domain, detected_framework, package_manager, dev_command, dev_port, status, created_at, updated_at FROM projects WHERE id = ?`, id).
		Scan(&p.ID, &p.Name, &p.Path, &p.Slug, &p.LocalDomain, &p.DetectedFramework,
			&p.PackageManager, &p.DevCommand, &p.DevPort, &status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Status = Status(status)

	scripts, err := r.scriptsByProject(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	p.Scripts = scripts
	return &p, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status Status) error {
	_, err := r.db.ExecContext(ctx, `UPDATE projects SET status = ?, updated_at = ? WHERE id = ?`, string(status), time.Now().UTC(), id)
	return err
}

func (r *Repository) scriptsByProject(ctx context.Context, id string) ([]Script, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, project_id, name, command, created_at FROM project_scripts WHERE project_id = ? ORDER BY name`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Script
	for rows.Next() {
		var s Script
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Name, &s.Command, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
