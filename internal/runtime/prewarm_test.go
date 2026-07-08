package runtime

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"orbit-app/internal/config"
	"orbit-app/internal/database"
	"orbit-app/internal/projects"
)

func TestPrewarmCandidatesByHourPattern(t *testing.T) {
	dir := t.TempDir()
	sqlDB, queries, err := database.Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	defer sqlDB.Close()

	repo := projects.NewRepository(sqlDB, queries)
	for _, id := range []string{"hot", "cold", "idle"} {
		if err := repo.Create(context.Background(), &projects.Project{ID: id, Name: id, Path: "/tmp/" + id, Slug: id}); err != nil {
			t.Fatalf("create %s: %v", id, err)
		}
	}
	m := New(repo, sqlDB, &config.Config{DataDir: dir}).(*manager)

	now := time.Now()
	batch := []Sample{}
	for d := 1; d <= 3; d++ {
		ts := now.AddDate(0, 0, -d).Unix()
		batch = append(batch, Sample{Ts: ts, ProjectID: "hot", ReqCount: 9, Status: projects.StatusRunning})
	}
	batch = append(batch, Sample{Ts: now.AddDate(0, 0, -1).Unix(), ProjectID: "cold", ReqCount: 3, Status: projects.StatusRunning})
	batch = append(batch, Sample{Ts: now.AddDate(0, 0, -2).Unix(), ProjectID: "idle", ReqCount: 0, Status: projects.StatusRunning})
	m.metrics.pushBatch(batch)

	got := m.prewarmCandidates(now)
	if len(got) != 1 || got[0] != "hot" {
		t.Fatalf("candidates = %v, want [hot]", got)
	}
}

func TestPrewarmTickDisabledByDefault(t *testing.T) {
	dir := t.TempDir()
	sqlDB, queries, err := database.Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	defer sqlDB.Close()

	m := New(projects.NewRepository(sqlDB, queries), sqlDB, &config.Config{DataDir: dir}).(*manager)
	if m.prewarmEnabled() {
		t.Fatal("prewarm should be off without a config")
	}
	m.prewarmTickOnce(time.Now())
}
