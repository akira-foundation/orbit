package projects

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"orbit-app/internal/database/db"
)

var ErrNotFound = errors.New("project not found")

type Repository struct {
	q   *db.Queries
	raw *sql.DB // needed for transactions
}

func NewRepository(sqlDB *sql.DB, q *db.Queries) *Repository {
	return &Repository{q: q, raw: sqlDB}
}

func (r *Repository) Create(ctx context.Context, p *Project) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}
	now := time.Now().UTC().Format(time.RFC3339)
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = StatusStopped
	}

	tx, err := r.raw.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	qtx := r.q.WithTx(tx)

	if _, err := qtx.CreateProject(ctx, db.CreateProjectParams{
		ID:                p.ID,
		Name:              p.Name,
		Path:              p.Path,
		Slug:              p.Slug,
		LocalDomain:       p.LocalDomain,
		DetectedFramework: p.DetectedFramework,
		PackageManager:    p.PackageManager,
		DevCommand:        p.DevCommand,
		DevPort:           int64(p.DevPort),
		NodeVersion:       p.NodeVersion,
		RuntimeKind:       string(p.RuntimeKind),
		PhpVersion:        p.PHPVersion,
		Status:            string(p.Status),
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}); err != nil {
		return err
	}

	for i := range p.Scripts {
		s := &p.Scripts[i]
		if s.ID == "" {
			s.ID = uuid.NewString()
		}
		s.ProjectID = p.ID
		s.CreatedAt = now
		if _, err := qtx.CreateScript(ctx, db.CreateScriptParams{
			ID:        s.ID,
			ProjectID: s.ProjectID,
			Name:      s.Name,
			Command:   s.Command,
			CreatedAt: s.CreatedAt,
		}); err != nil {
			return err
		}
	}

	if err := r.createProcesses(ctx, qtx, p, now); err != nil {
		return err
	}

	if p.LocalDomain != "" {
		if _, err := qtx.CreateDomain(ctx, db.CreateDomainParams{
			ID:         uuid.NewString(),
			ProjectID:  p.ID,
			Domain:     p.LocalDomain,
			TargetPort: int64(p.DevPort),
			Enabled:    1,
			CreatedAt:  now,
			UpdatedAt:  now,
		}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	if _, err := r.q.GetProject(ctx, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return r.q.DeleteProject(ctx, id)
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status Status) error {
	return r.q.UpdateProjectStatus(ctx, db.UpdateProjectStatusParams{
		Status:    string(status),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		ID:        id,
	})
}

func (r *Repository) InstalledHash(ctx context.Context, id string) (string, error) {
	var h string
	err := r.raw.QueryRowContext(ctx,
		`SELECT installed_hash FROM projects WHERE id = ?`, id,
	).Scan(&h)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return h, err
}

func (r *Repository) Secure(ctx context.Context, id string) (bool, error) {
	var v int
	err := r.raw.QueryRowContext(ctx,
		`SELECT secure FROM projects WHERE id = ?`, id,
	).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	return v == 1, err
}

func (r *Repository) SetSecure(ctx context.Context, id string, secure bool) error {
	v := 0
	if secure {
		v = 1
	}
	_, err := r.raw.ExecContext(ctx,
		`UPDATE projects SET secure = ?, updated_at = ? WHERE id = ?`,
		v, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

func (r *Repository) SetInstalledHash(ctx context.Context, id, hash string) error {
	_, err := r.raw.ExecContext(ctx,
		`UPDATE projects SET installed_hash = ?, updated_at = ? WHERE id = ?`,
		hash, time.Now().UTC().Format(time.RFC3339), id,
	)
	return err
}

func (r *Repository) List(ctx context.Context) ([]Project, error) {
	rows, err := r.q.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(rows))
	for i, row := range rows {
		out[i] = projectFromDB(row)
	}
	secureMap, err := r.secureMap(ctx)
	if err == nil {
		for i := range out {
			out[i].Secure = secureMap[out[i].ID]
		}
	}
	return out, nil
}

func (r *Repository) secureMap(ctx context.Context) (map[string]bool, error) {
	rows, err := r.raw.QueryContext(ctx, `SELECT id, secure FROM projects`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		var v int
		if err := rows.Scan(&id, &v); err == nil {
			out[id] = v == 1
		}
	}
	return out, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*Project, error) {
	row, err := r.q.GetProject(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p := projectFromDB(row)
	if secure, err := r.Secure(ctx, id); err == nil {
		p.Secure = secure
	}

	scripts, err := r.q.ListScriptsByProject(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, s := range scripts {
		p.Scripts = append(p.Scripts, Script{
			ID:        s.ID,
			ProjectID: s.ProjectID,
			Name:      s.Name,
			Command:   s.Command,
			CreatedAt: s.CreatedAt,
		})
	}

	procRows, err := r.q.ListProcessesByProject(ctx, id)
	if err != nil {
		return nil, err
	}
	for _, row := range procRows {
		p.Processes = append(p.Processes, processFromDB(row))
	}
	return &p, nil
}

func projectFromDB(row db.Project) Project {
	return Project{
		ID:                row.ID,
		Name:              row.Name,
		Path:              row.Path,
		Slug:              row.Slug,
		LocalDomain:       row.LocalDomain,
		DetectedFramework: row.DetectedFramework,
		PackageManager:    row.PackageManager,
		DevCommand:        row.DevCommand,
		DevPort:           int(row.DevPort),
		NodeVersion:       row.NodeVersion,
		RuntimeKind:       RuntimeKind(row.RuntimeKind),
		PHPVersion:        row.PhpVersion,
		Status:            Status(row.Status),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
