package runtime

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"orbit-app/internal/config"
	"orbit-app/internal/database"
	"orbit-app/internal/projects"
)

func TestMarkStartFailedRecordsSystemLog(t *testing.T) {
	dir := t.TempDir()
	sqlDB, queries, err := database.Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	defer sqlDB.Close()

	repo := projects.NewRepository(sqlDB, queries)
	proj := &projects.Project{ID: "proj-fail", Name: "app", Path: "/tmp/app", Slug: "app"}
	if err := repo.Create(context.Background(), proj); err != nil {
		t.Fatalf("repo.Create: %v", err)
	}

	m := New(repo, sqlDB, &config.Config{DataDir: dir}).(*manager)
	sess := newSession("proj-fail", 100)

	m.markStartFailed(sess, errors.New("acquire php 8.5.8: services: boom"))

	deadline := time.Now().Add(2 * time.Second)
	var logs []LogLine
	for time.Now().Before(deadline) {
		logs = m.LogsHistory("proj-fail", 0, 10)
		if len(logs) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(logs) == 0 {
		t.Fatal("expected markStartFailed to write a log entry, got none")
	}
	if logs[0].Text != "acquire php 8.5.8: services: boom" {
		t.Fatalf("log text = %q, want the failure error", logs[0].Text)
	}
}
