package projects

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"orbit-app/internal/database/db"
)

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
