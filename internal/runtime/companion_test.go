package runtime

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"orbit-app/internal/projects"
)

func TestFrontendProcessSpecFindsFrontendRole(t *testing.T) {
	proj := &projects.Project{Processes: []projects.ProcessSpec{
		{Role: projects.ProcessRoleBackend, Kind: projects.RuntimeKindPHPFPM},
		{Role: projects.ProcessRoleFrontend, Kind: projects.RuntimeKindCommand, Command: "npm run dev"},
	}}
	spec := frontendProcessSpec(proj)
	if spec == nil || spec.Command != "npm run dev" {
		t.Fatalf("expected frontend spec with npm run dev, got %+v", spec)
	}
}

func TestFrontendProcessSpecNilWhenBackendOnly(t *testing.T) {
	proj := &projects.Project{Processes: []projects.ProcessSpec{
		{Role: projects.ProcessRoleBackend, Kind: projects.RuntimeKindPHPFPM},
	}}
	if spec := frontendProcessSpec(proj); spec != nil {
		t.Fatalf("expected nil, got %+v", spec)
	}
}

func TestNeedsNodeInstallTrueWhenModulesMissing(t *testing.T) {
	dir := t.TempDir()
	if !needsNodeInstall(dir) {
		t.Fatal("expected install needed when node_modules is missing")
	}
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if needsNodeInstall(dir) {
		t.Fatal("expected no install needed once node_modules exists")
	}
}

func TestDetectCompanionPMFromLockfile(t *testing.T) {
	dir := t.TempDir()
	if got := detectCompanionPM(dir); got != "npm" {
		t.Fatalf("pm = %q, want npm default", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := detectCompanionPM(dir); got != "pnpm" {
		t.Fatalf("pm = %q, want pnpm", got)
	}
}

func TestStartCompanionSpawnsAndKillsFrontendProcess(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := &manager{emitter: nopEmitter{}}
	proj := &projects.Project{
		ID:   "proj-companion",
		Path: dir,
		Processes: []projects.ProcessSpec{
			{Role: projects.ProcessRoleFrontend, Kind: projects.RuntimeKindCommand, Command: "sleep 5"},
		},
	}
	sess := newSession(proj.ID, 100)

	m.startCompanion(context.Background(), sess, proj)

	deadline := time.Now().Add(2 * time.Second)
	for sess.CompanionPID() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	pid := sess.CompanionPID()
	if pid == 0 {
		t.Fatal("expected companion pid to be set")
	}
	if err := syscall.Kill(pid, 0); err != nil {
		t.Fatalf("expected companion process to be alive: %v", err)
	}
	if sess.companionKillFn == nil {
		t.Fatal("expected companionKillFn to be set")
	}

	sess.companionKillFn()
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("expected companion process to be killed")
}
