package runtime

import (
	"context"
	"testing"

	"orbit-app/internal/projects"
	"orbit-app/internal/services"
)

func TestNodeBinDirUsesSystemWhenPreferredAndAvailable(t *testing.T) {
	sys, ok := services.DetectSystemNode()
	if !ok {
		t.Skip("no system node on PATH for this test runner")
	}

	cfgStore := services.LoadRuntimesConfig(t.TempDir())
	if err := cfgStore.Save(services.RuntimesConfig{PreferSystemNode: true}); err != nil {
		t.Fatal(err)
	}

	m := &manager{runtimesConfig: cfgStore}
	proj := &projects.Project{NodeVersion: "22.23.1"}

	got, err := m.nodeBinDir(context.Background(), proj)
	if err != nil {
		t.Fatalf("nodeBinDir: %v", err)
	}
	if got != sys.BinDir {
		t.Fatalf("nodeBinDir = %q, want system bin dir %q", got, sys.BinDir)
	}
}

func TestNodeBinDirIgnoresSystemWhenNotPreferred(t *testing.T) {
	m := &manager{}
	proj := &projects.Project{}
	got, err := m.nodeBinDir(context.Background(), proj)
	if err != nil {
		t.Fatalf("nodeBinDir: %v", err)
	}
	if got != "" {
		t.Fatalf("nodeBinDir = %q, want empty for a non-Node project", got)
	}
}

func TestResolvePHPFPMBinUsesSystemWhenPreferredAndAvailable(t *testing.T) {
	sys, ok := services.DetectSystemPHP()
	if !ok {
		t.Skip("no system php+php-fpm pair on PATH for this test runner")
	}

	cfgStore := services.LoadRuntimesConfig(t.TempDir())
	if err := cfgStore.Save(services.RuntimesConfig{PreferSystemPHP: true}); err != nil {
		t.Fatal(err)
	}

	m := &manager{runtimesConfig: cfgStore}
	proj := &projects.Project{PHPVersion: "8.3.32"}

	got, isSystem, err := m.resolvePHPFPMBin(context.Background(), proj)
	if err != nil {
		t.Fatalf("resolvePHPFPMBin: %v", err)
	}
	if got != sys.PHPFPMPath {
		t.Fatalf("resolvePHPFPMBin = %q, want system php-fpm %q", got, sys.PHPFPMPath)
	}
	if !isSystem {
		t.Fatal("expected isSystem = true when using the detected system php-fpm")
	}
}

func TestResolvePHPFPMBinNotSystemWhenNotPreferred(t *testing.T) {
	m := &manager{}
	proj := &projects.Project{PHPVersion: "8.3.32"}

	_, isSystem, err := m.resolvePHPFPMBin(context.Background(), proj)
	if err == nil {
		t.Fatal("expected an error with no acquirer configured")
	}
	if isSystem {
		t.Fatal("expected isSystem = false when system PHP is not preferred")
	}
}
