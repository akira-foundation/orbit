package services

import (
	"context"
	"testing"

	"orbit-app/internal/database"
)

func newTestDB(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	sqlDB, _, err := database.Open(context.Background(), dir+"/test.db")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return NewStore(sqlDB)
}

func TestStoreEnableAndQuery(t *testing.T) {
	s := newTestDB(t)
	ctx := context.Background()

	if err := s.SetEnabled(ctx, "proj-a", "mailpit", true); err != nil {
		t.Fatalf("set enabled: %v", err)
	}
	if err := s.SetEnabled(ctx, "proj-b", "mailpit", true); err != nil {
		t.Fatalf("set enabled: %v", err)
	}

	engines, err := s.EnabledEngines(ctx, "proj-a")
	if err != nil || len(engines) != 1 || engines[0] != "mailpit" {
		t.Fatalf("enabled engines = %v err=%v", engines, err)
	}

	users, err := s.ProjectsUsing(ctx, "mailpit")
	if err != nil || len(users) != 2 {
		t.Fatalf("projects using = %v err=%v", users, err)
	}

	if err := s.SetEnabled(ctx, "proj-a", "mailpit", false); err != nil {
		t.Fatalf("disable: %v", err)
	}
	engines, _ = s.EnabledEngines(ctx, "proj-a")
	if len(engines) != 0 {
		t.Fatalf("expected no engines after disable, got %v", engines)
	}
}

func TestStoreBinaryRoundTrip(t *testing.T) {
	s := newTestDB(t)
	ctx := context.Background()

	if err := s.RecordBinary(ctx, "mailpit", "v1.20.0", "/p/mailpit", "abc123"); err != nil {
		t.Fatalf("record: %v", err)
	}
	b, ok, err := s.Binary(ctx, "mailpit", "v1.20.0")
	if err != nil || !ok || b.Hash != "abc123" || b.Path != "/p/mailpit" {
		t.Fatalf("binary = %+v ok=%v err=%v", b, ok, err)
	}
	_, ok, _ = s.Binary(ctx, "mailpit", "v9.9.9")
	if ok {
		t.Fatalf("expected missing binary")
	}
}
