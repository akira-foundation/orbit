package parked

import (
	"os"
	"path/filepath"
	"testing"
)

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func touch(t *testing.T, parts ...string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(parts...), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestIsProjectDirByManifestAndGit(t *testing.T) {
	dir := t.TempDir()
	if IsProjectDir(dir) {
		t.Fatal("empty dir must not be a project")
	}
	touch(t, dir, "package.json")
	if !IsProjectDir(dir) {
		t.Fatal("package.json must mark a project")
	}

	gitOnly := t.TempDir()
	mkdir(t, gitOnly, ".git")
	if !IsProjectDir(gitOnly) {
		t.Fatal(".git dir must mark a project")
	}
}

func TestDiscoverProjectsSkipsNoiseAndNonProjects(t *testing.T) {
	root := t.TempDir()
	app := mkdir(t, root, "shop")
	touch(t, app, "composer.json")
	mkdir(t, root, "node_modules")
	mkdir(t, root, ".cache")
	mkdir(t, root, "random-notes")

	found := DiscoverProjects(root)
	if len(found) != 1 || found[0] != app {
		t.Fatalf("found = %v", found)
	}
}

func TestStoreRoundTripAndDedup(t *testing.T) {
	dir := t.TempDir()
	s := LoadStore(dir)
	if err := s.AddFolder("/a"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFolder("/a"); err != nil {
		t.Fatal(err)
	}
	if err := s.AddFolder("/b"); err != nil {
		t.Fatal(err)
	}
	if got := s.Folders(); len(got) != 2 {
		t.Fatalf("folders = %v", got)
	}
	if err := s.MarkRegistered("/a/x", "id1"); err != nil {
		t.Fatal(err)
	}

	re := LoadStore(dir)
	if got := re.Folders(); len(got) != 2 {
		t.Fatalf("reloaded folders = %v", got)
	}
	if re.Registered()["/a/x"] != "id1" {
		t.Fatalf("registered = %v", re.Registered())
	}
	if err := re.RemoveFolder("/a"); err != nil {
		t.Fatal(err)
	}
	if got := re.Folders(); len(got) != 1 || got[0] != "/b" {
		t.Fatalf("after remove = %v", got)
	}
}
