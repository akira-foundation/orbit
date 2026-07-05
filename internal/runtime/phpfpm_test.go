package runtime

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPhpRunIDIsShortAndDeterministic(t *testing.T) {
	a := phpRunID("project-a")
	b := phpRunID("project-a")
	c := phpRunID("project-b")
	if a != b {
		t.Fatalf("phpRunID must be deterministic: %q != %q", a, b)
	}
	if a == c {
		t.Fatal("phpRunID must differ for different project IDs")
	}
	if len(a) != 12 {
		t.Fatalf("phpRunID length = %d, want 12", len(a))
	}
}

func TestWritePoolConfigContainsRequiredDirectives(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "f.sock")
	docRoot := filepath.Join(dir, "public")

	confPath, err := writePoolConfig(dir, sockPath, docRoot)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"listen = " + sockPath,
		"pm = ondemand",
		"chdir = " + docRoot,
		"daemonize = no",
	} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("pool config missing %q:\n%s", want, body)
		}
	}
}

func TestRemoveStaleSocketAllowsRebind(t *testing.T) {
	dir := t.TempDir()
	sockPath := filepath.Join(dir, "f.sock")

	first, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatal(err)
	}
	first.(*net.UnixListener).SetUnlinkOnClose(false)
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := net.Listen("unix", sockPath); err == nil {
		t.Fatal("expected bind to fail on a stale socket file left behind")
	}

	removeStaleSocket(sockPath)

	second, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("expected rebind to succeed after removeStaleSocket, got: %v", err)
	}
	_ = second.Close()
}

func TestWaitUnixSocketTimesOutWhenNeverCreated(t *testing.T) {
	dir := t.TempDir()
	err := waitUnixSocket(context.Background(), filepath.Join(dir, "never.sock"), 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error when socket never appears")
	}
}
