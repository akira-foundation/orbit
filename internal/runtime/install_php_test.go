package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"orbit-app/internal/projects"
)

func TestNeedsComposerInstallNoVendorDir(t *testing.T) {
	dir := t.TempDir()
	proj := &projects.Project{Path: dir}
	if !needsComposerInstall(proj, "") {
		t.Fatal("expected install needed when vendor/ is missing")
	}
}

func TestNeedsComposerInstallHashMismatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "composer.lock"), []byte(`{"content-hash":"abc"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	proj := &projects.Project{Path: dir}

	if !needsComposerInstall(proj, "stale-hash") {
		t.Fatal("expected install needed when lock hash differs from known hash")
	}

	hash, err := composerLockHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	if needsComposerInstall(proj, hash) {
		t.Fatal("expected no install needed when lock hash matches known hash")
	}
}

func TestNeedsComposerInstallNoLockfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	proj := &projects.Project{Path: dir}
	if needsComposerInstall(proj, "") {
		t.Fatal("expected no install needed when vendor/ exists and there is no lockfile to compare")
	}
}
