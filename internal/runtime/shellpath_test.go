package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLookPathInFindsExecutable(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bun")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got := lookPathIn("bun", "/nope:"+dir); got != bin {
		t.Fatalf("lookPathIn = %q want %q", got, bin)
	}
	if got := lookPathIn("/abs/bun", dir); got != "/abs/bun" {
		t.Fatalf("absolute passthrough = %q", got)
	}
	if got := lookPathIn("missing", dir); got != "missing" {
		t.Fatalf("missing fallback = %q", got)
	}
	nonExec := filepath.Join(dir, "data")
	_ = os.WriteFile(nonExec, []byte("x"), 0o644)
	if got := lookPathIn("data", dir); got != "data" {
		t.Fatalf("non-executable should be skipped, got %q", got)
	}
}

func TestMergePathListDedupsAndOrders(t *testing.T) {
	got := mergePathList("/usr/bin:/bin", "/opt/homebrew/bin:/usr/bin")
	if got != "/usr/bin:/bin:/opt/homebrew/bin" {
		t.Fatalf("merge = %q", got)
	}
}

func TestWithLoginPathAddsWhenMissing(t *testing.T) {
	env := withLoginPath([]string{"FOO=bar"})
	var pathVal string
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			pathVal = kv
		}
	}
	if pathVal == "" {
		t.Fatal("expected PATH to be added")
	}
}

func TestWithLoginPathMergesExisting(t *testing.T) {
	env := withLoginPath([]string{"PATH=/custom/bin"})
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			if !strings.Contains(kv, "/custom/bin") {
				t.Fatalf("lost existing PATH entry: %q", kv)
			}
			return
		}
	}
	t.Fatal("no PATH in env")
}
