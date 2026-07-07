package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAnalyzeDjangoProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "manage.py", "#!/usr/bin/env python\n")
	writeFile(t, dir, "requirements.txt", "Django==5.0\n")

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.RuntimeKind != "python" || a.Framework != "django" {
		t.Fatalf("kind=%q framework=%q", a.RuntimeKind, a.Framework)
	}
	if !strings.Contains(a.DevCommand, "manage.py runserver 127.0.0.1:$PORT") {
		t.Fatalf("dev command = %q", a.DevCommand)
	}
	if a.PackageManager != "pip" {
		t.Fatalf("pm = %q", a.PackageManager)
	}
	if a.PythonVersion == "" {
		t.Fatal("expected resolved python version")
	}
}

func TestAnalyzeFastAPIProjectUv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pyproject.toml", "[project]\nname='api'\nrequires-python='3.12'\ndependencies=['fastapi','uvicorn']\n")
	writeFile(t, dir, "uv.lock", "")
	writeFile(t, dir, "main.py", "app = object()\n")

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Framework != "fastapi" || a.PackageManager != "uv" {
		t.Fatalf("framework=%q pm=%q", a.Framework, a.PackageManager)
	}
	if !strings.HasPrefix(a.DevCommand, "uvicorn main:app") || !strings.Contains(a.DevCommand, "--port $PORT") {
		t.Fatalf("dev command = %q", a.DevCommand)
	}
	if a.PythonVersion != "3.12.13" {
		t.Fatalf("python version = %q", a.PythonVersion)
	}
}

func TestAnalyzeFlaskProjectPipenv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "Flask>=3.0\n")
	writeFile(t, dir, "Pipfile", "[packages]\nflask = \"*\"\n")

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.Framework != "flask" || a.PackageManager != "pipenv" {
		t.Fatalf("framework=%q pm=%q", a.Framework, a.PackageManager)
	}
	if !strings.HasPrefix(a.DevCommand, "flask run") {
		t.Fatalf("dev command = %q", a.DevCommand)
	}
}

func TestPythonVersionFileWins(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "requests\n")
	writeFile(t, dir, ".python-version", "3.11\n")

	a, err := New().Analyze(dir)
	if err != nil {
		t.Fatal(err)
	}
	if a.PythonVersion != "3.11.15" {
		t.Fatalf("python version = %q", a.PythonVersion)
	}
}
