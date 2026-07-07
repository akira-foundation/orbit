package runtime

import "testing"

func TestReadWriteProjectEnv(t *testing.T) {
	dir := t.TempDir()
	in := map[string]string{"APP_NAME": "orbit", "DB_PORT": "5433"}
	if err := WriteProjectEnv(dir, in); err != nil {
		t.Fatal(err)
	}
	out, err := ReadProjectEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	if out["APP_NAME"] != "orbit" || out["DB_PORT"] != "5433" {
		t.Fatalf("roundtrip = %v", out)
	}
}

func TestReadProjectEnvMissingFileIsEmpty(t *testing.T) {
	out, err := ReadProjectEnv(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty, got %v", out)
	}
}
