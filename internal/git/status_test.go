package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStatusNonRepo(t *testing.T) {
	info := Status(context.Background(), t.TempDir())
	if info.Repo {
		t.Fatalf("expected non-repo, got %+v", info)
	}
}

func TestStatusCleanThenDirty(t *testing.T) {
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	run("init", "-b", "main")
	run("config", "user.email", "t@t.dev")
	run("config", "user.name", "t")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "--no-verify", "-m", "chore: init")

	info := Status(context.Background(), dir)
	if !info.Repo || info.Branch != "main" || info.Dirty != 0 {
		t.Fatalf("clean repo = %+v", info)
	}

	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	if d := Status(context.Background(), dir); d.Dirty != 1 {
		t.Fatalf("dirty = %d, want 1", d.Dirty)
	}
}
