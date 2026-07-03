package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNodeCatalogEntries(t *testing.T) {
	wantVersions := map[string]string{
		"node-24": "24.18.0",
		"node-22": "22.23.1",
		"node-20": "20.20.2",
	}
	byID := map[string]Engine{}
	for _, e := range nodeEngines() {
		byID[e.ID] = e
	}
	for id, version := range wantVersions {
		e, ok := byID[id]
		if !ok {
			t.Fatalf("%s missing from node engines", id)
		}
		if e.Version != version || e.Family != "node" || !e.ExtractTree {
			t.Fatalf("%s = version %q family %q tree %v", id, e.Version, e.Family, e.ExtractTree)
		}
		if len(e.Platforms) != 4 {
			t.Fatalf("%s platforms = %d", id, len(e.Platforms))
		}
		for key, p := range e.Platforms {
			if len(p.SHA256) != 64 {
				t.Fatalf("%s %s missing pinned sha256", id, key)
			}
			if !strings.Contains(p.ArchiveBinaryPath, "/bin/node") {
				t.Fatalf("%s %s bad binary path %q", id, key, p.ArchiveBinaryPath)
			}
		}
	}
}

func TestNodeVersionsNewestFirst(t *testing.T) {
	got := NodeVersions()
	want := []string{"24.18.0", "22.23.1", "20.20.2"}
	if len(got) != len(want) {
		t.Fatalf("NodeVersions() = %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("NodeVersions()[%d] = %q, want %q", i, got[i], v)
		}
	}
}

func TestNodeEnginesExcludedFromGeneralCatalog(t *testing.T) {
	if _, ok := ResolveEngine("node-22"); ok {
		t.Fatal("node engines must not be resolvable via the general services catalog")
	}
	for _, e := range Catalog() {
		if e.Family == "node" {
			t.Fatalf("Catalog() must not include node engines, found %s", e.ID)
		}
	}
}

func TestNodeEngineForVersion(t *testing.T) {
	e, ok := NodeEngineForVersion("22.23.1")
	if !ok {
		t.Fatal("expected engine for 22.23.1")
	}
	if e.ID != "node-22" {
		t.Fatalf("NodeEngineForVersion(22.23.1).ID = %q, want node-22", e.ID)
	}
	if _, ok := NodeEngineForVersion("1.0.0"); ok {
		t.Fatal("expected no engine for unknown version 1.0.0")
	}
}

func TestInstallNodeVersionUnknownVersion(t *testing.T) {
	store := newTestDB(t)
	acq := NewAcquirer(store, t.TempDir())
	if err := InstallNodeVersion(context.Background(), acq, "1.0.0"); err == nil {
		t.Fatal("expected error installing unknown version")
	}
}

func TestNodeVersionsInfoAndRemove(t *testing.T) {
	store := newTestDB(t)
	base := t.TempDir()
	acq := NewAcquirer(store, base)
	ctx := context.Background()

	e, ok := NodeEngineForVersion("22.23.1")
	if !ok {
		t.Fatal("expected engine for 22.23.1")
	}
	p, ok := e.CurrentPlatform()
	if !ok {
		t.Skip("no bundled platform for this test runner's GOOS/GOARCH")
	}
	binPath := filepath.Join(base, e.ID, e.Version, filepath.FromSlash(p.ArchiveBinaryPath))
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	infos := NodeVersionsInfo(ctx, acq)
	var found bool
	for _, info := range infos {
		if info.Version != "22.23.1" {
			if info.Installed {
				t.Fatalf("%s should not be installed", info.Version)
			}
			continue
		}
		found = true
		if !info.Installed {
			t.Fatal("22.23.1 should be installed")
		}
		if info.DiskBytes <= 0 {
			t.Fatal("22.23.1 should report nonzero disk usage")
		}
	}
	if !found {
		t.Fatal("22.23.1 missing from NodeVersionsInfo")
	}

	if err := RemoveNodeVersion(ctx, acq, "22.23.1"); err != nil {
		t.Fatalf("RemoveNodeVersion: %v", err)
	}
	if acq.IsInstalled(ctx, e) {
		t.Fatal("expected 22.23.1 to be uninstalled after RemoveNodeVersion")
	}
	if err := RemoveNodeVersion(ctx, acq, "1.0.0"); err == nil {
		t.Fatal("expected error removing unknown version")
	}
}
