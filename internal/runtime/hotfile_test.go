package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"orbit-app/internal/projects"
)

func TestWaitHotFileReturnsOnceFileExists(t *testing.T) {
	dir := t.TempDir()
	hotPath := filepath.Join(dir, "hot")

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = os.WriteFile(hotPath, []byte("http://[::1]:5173"), 0o644)
	}()

	if err := waitHotFile(context.Background(), dir, 2*time.Second); err != nil {
		t.Fatalf("expected hot file to appear in time, got: %v", err)
	}
}

func TestWaitHotFileTimesOutWhenNeverWritten(t *testing.T) {
	dir := t.TempDir()
	if err := waitHotFile(context.Background(), dir, 100*time.Millisecond); err == nil {
		t.Fatal("expected a timeout error when hot never appears")
	}
}

func TestStartCompanionAndWaitNoopWithoutFrontendSpec(t *testing.T) {
	m := &manager{emitter: nopEmitter{}}
	proj := &projects.Project{ID: "no-frontend", Path: t.TempDir()}
	sess := newSession(proj.ID, 10)

	done := make(chan struct{})
	go func() {
		m.startCompanionAndWait(context.Background(), sess, proj)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected startCompanionAndWait to return immediately with no frontend spec")
	}
}
