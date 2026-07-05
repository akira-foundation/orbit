package analyzer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeLaravelProject(t *testing.T) {
	dir := t.TempDir()
	composerJSON := `{"name":"acme/app","require":{"php":"^8.3","laravel/framework":"^11.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Framework != "laravel" {
		t.Fatalf("framework = %q, want laravel", a.Framework)
	}
	if a.RuntimeKind != "php-fpm" {
		t.Fatalf("runtimeKind = %q, want php-fpm", a.RuntimeKind)
	}
	if a.PHPVersion == "" {
		t.Fatal("expected a resolved php version")
	}
	if a.PackageManager != "composer" {
		t.Fatalf("packageManager = %q, want composer", a.PackageManager)
	}
}

func TestAnalyzeLaravelUsesFolderNameNotComposerName(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "nosferry.com")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	composerJSON := `{"name":"nosferry/nosferry.com","require":{"php":"^8.5","laravel/framework":"^11.0"}}`
	if err := os.WriteFile(filepath.Join(appDir, "composer.json"), []byte(composerJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := New().Analyze(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "nosferry.com" {
		t.Fatalf("name = %q, want folder name nosferry.com", a.Name)
	}
}

func TestAnalyzeNodeUsesFolderNameNotPackageName(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "my-app")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "package.json"), []byte(`{"name":"@acme/totally-different"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := New().Analyze(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "my-app" {
		t.Fatalf("name = %q, want folder name my-app", a.Name)
	}
}

func TestAnalyzeNodeProjectDefaultsToCommandRuntime(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"app"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.RuntimeKind != "command" {
		t.Fatalf("runtimeKind = %q, want command", a.RuntimeKind)
	}
	if a.PHPVersion != "" {
		t.Fatalf("expected no php version for a Node project, got %q", a.PHPVersion)
	}
}
