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
