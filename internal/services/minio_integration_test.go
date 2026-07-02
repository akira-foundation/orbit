package services

import (
	"context"
	"testing"
	"time"

	"orbit-app/internal/services/miniostore"
)

func TestRealMinioEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("network + heavy")
	}
	e, ok := ResolveEngine("minio")
	if !ok {
		t.Fatal("minio missing")
	}

	store := newTestDB(t)
	acq := NewAcquirer(store, t.TempDir())
	resolver := NewResolver(store, AllowAll())
	m := NewManager(acq, resolver, LoadConfig(t.TempDir()), t.TempDir())
	m.reap = func(string) {}
	t.Cleanup(m.StopAll)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if err := m.Acquire(ctx, "minio", "proj-x"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if got := m.Status("minio"); got.Status != "running" {
		t.Fatalf("status = %+v", got)
	}

	if err := miniostore.EnsureBucket(ctx, e.Bind, e.Port, "integration-app"); err != nil {
		t.Fatalf("ensure bucket: %v", err)
	}
	if err := miniostore.EnsureBucket(ctx, e.Bind, e.Port, "integration-app"); err != nil {
		t.Fatalf("ensure bucket idempotent: %v", err)
	}

	m.Release("minio", "proj-x")
}
