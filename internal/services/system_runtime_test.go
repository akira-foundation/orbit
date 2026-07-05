package services

import "testing"

func TestDetectSystemNodeShapeWhenPresent(t *testing.T) {
	got, ok := DetectSystemNode()
	if !ok {
		t.Skip("no system node on PATH for this test runner")
	}
	if got.Version == "" || got.BinDir == "" {
		t.Fatalf("incomplete detection: %+v", got)
	}
}

func TestDetectSystemPHPShapeWhenPresent(t *testing.T) {
	got, ok := DetectSystemPHP()
	if !ok {
		t.Skip("no system php+php-fpm pair on PATH for this test runner")
	}
	if got.Version == "" || got.PHPPath == "" || got.PHPFPMPath == "" {
		t.Fatalf("incomplete detection: %+v", got)
	}
}

func TestDetectSystemPHPFPMVersionMatchesPHPVersion(t *testing.T) {
	got, ok := DetectSystemPHP()
	if !ok {
		t.Skip("no system php+php-fpm pair on PATH for this test runner")
	}
	fpmVersion, ok := phpVersionOf(got.PHPFPMPath)
	if !ok {
		t.Fatalf("could not determine version of resolved php-fpm %q", got.PHPFPMPath)
	}
	if fpmVersion != got.Version {
		t.Fatalf("resolved php-fpm %q is version %q, want it to match php's version %q",
			got.PHPFPMPath, fpmVersion, got.Version)
	}
}

func TestRuntimesConfigDefaultsToNotPreferringSystem(t *testing.T) {
	store := LoadRuntimesConfig(t.TempDir())
	if store.PreferSystemNode() || store.PreferSystemPHP() {
		t.Fatal("expected bundled runtimes to be preferred by default")
	}
}

func TestRuntimesConfigSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	store := LoadRuntimesConfig(dir)
	if err := store.Save(RuntimesConfig{PreferSystemNode: true, PreferSystemPHP: true}); err != nil {
		t.Fatal(err)
	}

	reloaded := LoadRuntimesConfig(dir)
	if !reloaded.PreferSystemNode() || !reloaded.PreferSystemPHP() {
		t.Fatalf("config did not persist: %+v", reloaded.Get())
	}
}
