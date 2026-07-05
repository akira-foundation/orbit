package projects

import (
	"context"

	"github.com/google/uuid"

	"orbit-app/internal/database/db"
)

func (r *Repository) createProcesses(ctx context.Context, qtx *db.Queries, p *Project, now string) error {
	processes := p.Processes
	if len(processes) == 0 {
		processes = []ProcessSpec{{
			Role:       ProcessRoleBackend,
			Kind:       p.RuntimeKind,
			Command:    p.DevCommand,
			Port:       p.DevPort,
			PHPVersion: p.PHPVersion,
		}}
	}
	for i := range processes {
		ps := &processes[i]
		if ps.ID == "" {
			ps.ID = uuid.NewString()
		}
		ps.ProjectID = p.ID
		ps.Order = i
		ps.CreatedAt = now
		if _, err := qtx.CreateProcess(ctx, db.CreateProcessParams{
			ID:         ps.ID,
			ProjectID:  ps.ProjectID,
			Role:       string(ps.Role),
			Kind:       string(ps.Kind),
			WorkDir:    ps.WorkDir,
			Command:    ps.Command,
			Port:       int64(ps.Port),
			PhpVersion: ps.PHPVersion,
			SortOrder:  int64(ps.Order),
			CreatedAt:  ps.CreatedAt,
		}); err != nil {
			return err
		}
	}
	p.Processes = processes
	return nil
}

func processFromDB(row db.ProjectProcess) ProcessSpec {
	return ProcessSpec{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		Role:       ProcessRole(row.Role),
		Kind:       RuntimeKind(row.Kind),
		WorkDir:    row.WorkDir,
		Command:    row.Command,
		Port:       int(row.Port),
		PHPVersion: row.PhpVersion,
		Order:      int(row.SortOrder),
		CreatedAt:  row.CreatedAt,
	}
}
