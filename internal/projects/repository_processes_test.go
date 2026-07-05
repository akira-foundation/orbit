package projects

import (
	"context"
	"path/filepath"
	"testing"

	"orbit-app/internal/database"
)

func newTestRepository(t *testing.T) *Repository {
	t.Helper()
	dir := t.TempDir()
	sqlDB, queries, err := database.Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewRepository(sqlDB, queries)
}

func TestRepositoryCreateDefaultsToSingleBackendProcess(t *testing.T) {
	repo := newTestRepository(t)
	p := &Project{
		Name:        "app",
		Path:        "/tmp/app",
		Slug:        "app",
		RuntimeKind: RuntimeKindPHPFPM,
		PHPVersion:  "8.3",
	}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Processes) != 1 {
		t.Fatalf("processes = %d, want 1", len(got.Processes))
	}
	if got.Processes[0].Role != ProcessRoleBackend || got.Processes[0].Kind != RuntimeKindPHPFPM {
		t.Fatalf("processes[0] = %+v, want backend/php-fpm", got.Processes[0])
	}
}

func TestRepositoryCreatePersistsExplicitProcessList(t *testing.T) {
	repo := newTestRepository(t)
	p := &Project{
		Name:        "inertia-app",
		Path:        "/tmp/inertia-app",
		Slug:        "inertia-app",
		RuntimeKind: RuntimeKindPHPFPM,
		Processes: []ProcessSpec{
			{Role: ProcessRoleBackend, Kind: RuntimeKindPHPFPM, PHPVersion: "8.3"},
			{Role: ProcessRoleFrontend, Kind: RuntimeKindCommand, Command: "npm run dev"},
		},
	}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.Get(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Processes) != 2 {
		t.Fatalf("processes = %d, want 2", len(got.Processes))
	}
	if got.Processes[1].Role != ProcessRoleFrontend || got.Processes[1].Command != "npm run dev" {
		t.Fatalf("processes[1] = %+v, want frontend npm run dev", got.Processes[1])
	}
}
