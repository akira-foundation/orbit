package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeJSON(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzeInertiaDetectsFrontendCompanion(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "composer.json"),
		`{"name":"acme/app","require":{"php":"^8.3","laravel/framework":"^11.0"}}`)
	writeJSON(t, filepath.Join(dir, "package.json"),
		`{"name":"app","scripts":{"dev":"vite"},"devDependencies":{"vite":"^5.0.0","laravel-vite-plugin":"^1.0.0"}}`)

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Framework != "laravel-inertia" {
		t.Fatalf("framework = %q, want laravel-inertia", a.Framework)
	}
	if len(a.Processes) != 2 {
		t.Fatalf("processes = %d, want 2 (backend + frontend)", len(a.Processes))
	}
	if a.Processes[0].Role != ProcessRoleBackend || a.Processes[0].Kind != "php-fpm" {
		t.Fatalf("processes[0] = %+v, want backend/php-fpm", a.Processes[0])
	}
	if a.Processes[1].Role != ProcessRoleFrontend || a.Processes[1].Command == "" {
		t.Fatalf("processes[1] = %+v, want frontend with a command", a.Processes[1])
	}
}

func TestAnalyzeLaravelWithoutViteHasNoFrontendCompanion(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "composer.json"),
		`{"name":"acme/app","require":{"php":"^8.3","laravel/framework":"^11.0"}}`)

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Framework != "laravel" {
		t.Fatalf("framework = %q, want laravel", a.Framework)
	}
	if len(a.Processes) != 1 {
		t.Fatalf("processes = %d, want 1 (backend only)", len(a.Processes))
	}
}

func TestAnalyzeMonorepoResolvesBackendAndFrontend(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "backend", "composer.json"),
		`{"name":"acme/api","require":{"php":"^8.3","laravel/framework":"^11.0"}}`)
	writeJSON(t, filepath.Join(dir, "frontend", "package.json"),
		`{"name":"web","scripts":{"dev":"vite"},"dependencies":{"vite":"^5.0.0"}}`)

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Framework != "monorepo" {
		t.Fatalf("framework = %q, want monorepo", a.Framework)
	}
	if a.AmbiguousLayout {
		t.Fatal("expected unambiguous layout")
	}
	if len(a.Processes) != 2 {
		t.Fatalf("processes = %d, want 2", len(a.Processes))
	}
	if a.Processes[0].WorkDir != "backend" {
		t.Fatalf("backend workDir = %q, want backend", a.Processes[0].WorkDir)
	}
	if a.Processes[1].WorkDir != "frontend" {
		t.Fatalf("frontend workDir = %q, want frontend", a.Processes[1].WorkDir)
	}
}

func TestAnalyzeMonorepoAmbiguousWithMultipleBackends(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "api", "composer.json"),
		`{"name":"acme/api","require":{"laravel/framework":"^11.0"}}`)
	writeJSON(t, filepath.Join(dir, "admin", "composer.json"),
		`{"name":"acme/admin","require":{"laravel/framework":"^11.0"}}`)

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !a.AmbiguousLayout {
		t.Fatal("expected ambiguous layout with two backend candidates")
	}
	if len(a.Processes) != 0 {
		t.Fatalf("expected no auto-wired processes, got %d", len(a.Processes))
	}
}

func TestAnalyzeMonorepoNoBackendCandidateReturnsNotFoundError(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, filepath.Join(dir, "docs", "readme.txt"), "hello")

	if _, err := New().Analyze(dir); err == nil {
		t.Fatal("expected an error when no project config is found anywhere")
	}
}
