package runtime

import (
	"os"
	"path/filepath"
	"testing"

	"orbit-app/internal/projects"
)

func TestSubstitutePort(t *testing.T) {
	cases := map[string]string{
		"uvicorn main:app --port $PORT":     "uvicorn main:app --port 4321",
		"python manage.py runserver :$PORT": "python manage.py runserver :4321",
		"flask run --port ${PORT}":          "flask run --port 4321",
		"node server.js":                    "node server.js",
	}
	for in, want := range cases {
		if got := substitutePort(in, 4321); got != want {
			t.Fatalf("substitutePort(%q) = %q want %q", in, got, want)
		}
	}
}

func TestPythonInstallCommandPrefersPip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "requirements.txt"), []byte("flask\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proj := &projects.Project{Path: dir, PackageManager: "pip"}
	cmd := pythonInstallCommand(proj)
	if len(cmd) == 0 || cmd[0] != "python" || cmd[len(cmd)-1] != "requirements.txt" {
		t.Fatalf("cmd = %v", cmd)
	}
}

func TestPythonInstallCommandPyprojectFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname='x'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	proj := &projects.Project{Path: dir, PackageManager: "pip"}
	cmd := pythonInstallCommand(proj)
	if len(cmd) == 0 || cmd[len(cmd)-1] != "." {
		t.Fatalf("cmd = %v", cmd)
	}
}

func TestResolvePythonBinUsesVenvInterpreter(t *testing.T) {
	venv := t.TempDir()
	py := filepath.Join(venv, "python")
	if err := os.WriteFile(py, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := resolvePythonBin("python", venv); got != py {
		t.Fatalf("resolve = %q want %q", got, py)
	}
	if got := resolvePythonBin("python", ""); got != "python" {
		t.Fatalf("no venv = %q", got)
	}
	if got := resolvePythonBin("uv", venv); got != "uv" {
		t.Fatalf("non-python = %q", got)
	}
	if got := resolvePythonBin("python", t.TempDir()); got != "python" {
		t.Fatalf("venv without python = %q", got)
	}
}

func TestNeedsPythonInstallWhenVenvMissing(t *testing.T) {
	dir := t.TempDir()
	proj := &projects.Project{Path: dir, PythonVersion: "3.12.13"}
	if !needsPythonInstall(proj, "") {
		t.Fatal("expected install needed without .venv")
	}
}
