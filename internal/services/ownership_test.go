package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestProcessExecutableResolvesOwnPath(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	exe := processExecutable(os.Getpid())
	if exe == "" {
		t.Fatal("expected non-empty executable path")
	}
	if filepath.Base(exe) != filepath.Base(self) {
		t.Fatalf("exe = %q, want basename matching %q", exe, self)
	}
}

func spawnBoundListener(t *testing.T, port int) *exec.Cmd {
	t.Helper()
	script := fmt.Sprintf(`
import socket, time
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
s.bind(("127.0.0.1", %d))
s.listen(1)
time.sleep(30)
`, port)
	cmd := exec.Command("python3", "-c", script)
	if err := cmd.Start(); err != nil {
		t.Skipf("python3 not available: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	time.Sleep(300 * time.Millisecond)
	return cmd
}

func TestPortOwnedPIDsMatchesPrefix(t *testing.T) {
	port := freePort(t)
	cmd := spawnBoundListener(t, port)

	exe := processExecutable(cmd.Process.Pid)
	if exe == "" {
		t.Skip("could not resolve python3 executable path")
	}

	pids := portOwnedPIDs("127.0.0.1", port, filepath.Dir(exe))
	found := false
	for _, p := range pids {
		if p == cmd.Process.Pid {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected pid %d among owned pids, got %v", cmd.Process.Pid, pids)
	}
}

func TestPortOwnedPIDsRejectsMismatchedPrefix(t *testing.T) {
	port := freePort(t)
	spawnBoundListener(t, port)

	pids := portOwnedPIDs("127.0.0.1", port, "/definitely/not/a/real/orbit/services/path")
	if len(pids) != 0 {
		t.Fatalf("expected no owned pids for mismatched prefix, got %v", pids)
	}
}
