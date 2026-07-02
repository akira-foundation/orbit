package services

import (
	"context"
	"testing"
	"time"

	"orbit-app/internal/services/postgres"
)

func TestRealPostgres18EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("network + heavy")
	}
	e, ok := ResolveEngine("postgres-18")
	if !ok {
		t.Fatal("postgres-18 missing")
	}

	store := newTestDB(t)
	acq := NewAcquirer(store, t.TempDir())
	resolver := NewResolver(store, AllowAll())
	m := NewManager(acq, resolver, LoadConfig(t.TempDir()), t.TempDir())
	m.reap = func(string) {}
	t.Cleanup(m.StopAll)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := m.Acquire(ctx, "postgres-18", "proj-x"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if got := m.Status("postgres-18"); got.Status != "running" {
		t.Fatalf("status = %+v", got)
	}

	if err := postgres.EnsureDatabase(ctx, e.Bind, e.Port, "integration_app"); err != nil {
		t.Fatalf("ensure database: %v", err)
	}
	if err := postgres.EnsureDatabase(ctx, e.Bind, e.Port, "integration_app"); err != nil {
		t.Fatalf("ensure database idempotent: %v", err)
	}

	m.Release("postgres-18", "proj-x")
}
