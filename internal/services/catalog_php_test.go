package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPHPVersionsAndEngineForVersion(t *testing.T) {
	versions := PHPVersions()
	if len(versions) == 0 {
		t.Fatal("expected at least one bundled php version")
	}
	e, ok := PHPEngineForVersion(versions[0])
	if !ok {
		t.Fatalf("expected engine for version %s", versions[0])
	}
	if e.Family != "php" || !e.ExtractTree {
		t.Fatalf("engine = %+v, want family=php extractTree=true", e)
	}
	if _, ok := PHPEngineForVersion("0.0.0"); ok {
		t.Fatal("expected no engine for unknown version")
	}
}

func TestPHPAndComposerEnginesExcludedFromGeneralCatalog(t *testing.T) {
	if _, ok := ResolveEngine("php-8.3"); ok {
		t.Fatal("php engines must not be resolvable via the general services catalog")
	}
	if _, ok := ResolveEngine("composer"); ok {
		t.Fatal("composer must not be resolvable via the general services catalog")
	}
	for _, e := range Catalog() {
		if e.Family == "php" || e.Family == "composer" {
			t.Fatalf("Catalog() must not include toolchain engines, found %s", e.ID)
		}
	}
}

func TestPHPVersionsInfoAndRemove(t *testing.T) {
	store := newTestDB(t)
	base := t.TempDir()
	acq := NewAcquirer(store, base)
	ctx := context.Background()

	versions := PHPVersions()
	if len(versions) == 0 {
		t.Fatal("expected at least one bundled php version")
	}
	target := versions[0]
	e, ok := PHPEngineForVersion(target)
	if !ok {
		t.Fatalf("expected engine for %s", target)
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

	infos := PHPVersionsInfo(ctx, acq)
	var found bool
	for _, info := range infos {
		if info.Version != target {
			continue
		}
		found = true
		if !info.Installed || info.DiskBytes <= 0 {
			t.Fatalf("%s = %+v, want installed with nonzero disk usage", target, info)
		}
	}
	if !found {
		t.Fatalf("%s missing from PHPVersionsInfo", target)
	}

	if err := RemovePHPVersion(ctx, acq, target); err != nil {
		t.Fatalf("RemovePHPVersion: %v", err)
	}
	if acq.IsInstalled(ctx, e) {
		t.Fatalf("expected %s to be uninstalled", target)
	}
	if err := RemovePHPVersion(ctx, acq, "0.0.0"); err == nil {
		t.Fatal("expected error removing unknown version")
	}
}

func TestInstallPHPVersionUnknownVersion(t *testing.T) {
	acq := NewAcquirer(newTestDB(t), t.TempDir())
	if err := InstallPHPVersion(context.Background(), acq, "0.0.0"); err == nil {
		t.Fatal("expected error installing unknown version")
	}
}

func TestComposerEngineHasPinnedChecksum(t *testing.T) {
	e := ComposerEngine()
	if e.Version == "" {
		t.Fatal("expected a pinned composer version")
	}
	p, ok := e.CurrentPlatform()
	if !ok {
		t.Skip("no bundled platform for this test runner's GOOS/GOARCH")
	}
	if len(p.SHA256) != 64 {
		t.Fatalf("composer checksum length = %d, want 64", len(p.SHA256))
	}
}
