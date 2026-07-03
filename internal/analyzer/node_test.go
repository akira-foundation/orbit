package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"orbit-app/internal/services"
)

func TestResolveNodeVersion_NvmrcTakesPriority(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".nvmrc"), "v20\n")
	mustWrite(t, filepath.Join(dir, ".node-version"), "22")

	if got, want := resolveNodeVersion(dir, packageJSON{}), "20.20.2"; got != want {
		t.Errorf("resolveNodeVersion() = %q, want %q", got, want)
	}
}

func TestResolveNodeVersion_NodeVersionFileWhenNoNvmrc(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, ".node-version"), "22.23.1")

	if got, want := resolveNodeVersion(dir, packageJSON{}), "22.23.1"; got != want {
		t.Errorf("resolveNodeVersion() = %q, want %q", got, want)
	}
}

func TestResolveNodeVersion_EnginesFieldWhenNoVersionFile(t *testing.T) {
	dir := t.TempDir()
	pkg := packageJSON{Engines: map[string]string{"node": ">=20.0.0 <22.0.0"}}

	if got, want := resolveNodeVersion(dir, pkg), "20.20.2"; got != want {
		t.Errorf("resolveNodeVersion() = %q, want %q", got, want)
	}
}

func TestResolveNodeVersion_NoSignalFallsBackToNewest(t *testing.T) {
	dir := t.TempDir()
	available := services.NodeVersions()
	if len(available) == 0 {
		t.Fatal("expected at least one bundled node version in the catalog")
	}
	if got, want := resolveNodeVersion(dir, packageJSON{}), available[0]; got != want {
		t.Errorf("resolveNodeVersion() = %q, want newest %q", got, want)
	}
}

func TestResolveNodeVersion_UnparseableFallsBackToNewest(t *testing.T) {
	dir := t.TempDir()
	pkg := packageJSON{Engines: map[string]string{"node": "not-a-real-constraint"}}
	available := services.NodeVersions()

	if got, want := resolveNodeVersion(dir, pkg), available[0]; got != want {
		t.Errorf("resolveNodeVersion() = %q, want newest %q", got, want)
	}
}

func TestResolveNodeVersion_UnsatisfiableConstraintFallsBackToNewest(t *testing.T) {
	dir := t.TempDir()
	pkg := packageJSON{Engines: map[string]string{"node": ">=99.0.0"}}
	available := services.NodeVersions()

	if got, want := resolveNodeVersion(dir, pkg), available[0]; got != want {
		t.Errorf("resolveNodeVersion() = %q, want newest %q", got, want)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
