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

// Repository wraps the SQLC-generated *db.Queries with domain-level logic:
// transaction handling, model mapping, and error translation.
type Repository struct {
	q   *db.Queries
	raw *sql.DB // needed for transactions
}

func NewRepository(sqlDB *sql.DB, q *db.Queries) *Repository {
	return &Repository{q: q, raw: sqlDB}
}

// ─── writes ──────────────────────────────────────────────────────────────────

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

func (r *Repository) ListDomains(ctx context.Context) ([]Domain, error) {
	rows, err := r.q.ListEnabledDomains(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Domain, len(rows))
	for i, row := range rows {
		out[i] = domainFromDB(row)
	}
	return out, nil
}

func (r *Repository) GetDomain(ctx context.Context, host string) (*Domain, error) {
	row, err := r.q.GetDomain(ctx, host)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d := domainFromDB(row)
	return &d, nil
}

func (r *Repository) UpdateDomainPort(ctx context.Context, id string, port int) error {
	return r.q.UpdateDomainPort(ctx, db.UpdateDomainPortParams{
		TargetPort: int64(port),
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		ID:         id,
	})
}

func domainFromDB(row db.ProjectDomain) Domain {
	return Domain{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Domain:     row.Domain,
		TargetPort: int(row.TargetPort),
		Enabled:    row.Enabled == 1,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
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

// ─── reads ────────────────────────────────────────────────────────────────────

func (r *Repository) List(ctx context.Context) ([]Project, error) {
	rows, err := r.q.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Project, len(rows))
	for i, row := range rows {
		out[i] = projectFromDB(row)
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
	return &p, nil
}

// ─── helpers ─────────────────────────────────────────────────────────────────

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
		Status:            Status(row.Status),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}
