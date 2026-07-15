package config

import (
	"path/filepath"
	"testing"
)

func TestLoadUsesDataDirOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("ORBIT_DATA_DIR", dir)

	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.DataDir != dir {
		t.Fatalf("DataDir = %q want %q", c.DataDir, dir)
	}
	if c.DBPath != filepath.Join(dir, "orbit.db") {
		t.Fatalf("DBPath = %q", c.DBPath)
	}
}

func TestDefaultDataDirName(t *testing.T) {
	t.Setenv("ORBIT_DATA_DIR", "")
	dir, err := dataDir()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != dataDirName {
		t.Fatalf("default dir base = %q want %q", filepath.Base(dir), dataDirName)
	}
}
