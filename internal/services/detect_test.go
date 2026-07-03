package services

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDetectTokensFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"dependencies": {"pg": "^8.0.0", "nodemailer": "^6.0.0"},
		"devDependencies": {"@aws-sdk/client-s3": "^3.0.0"}
	}`)

	got := DetectTokens(dir)
	want := map[string]bool{"postgres": true, "mailpit": true, "minio": true}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, tok := range got {
		if !want[tok] {
			t.Fatalf("unexpected token %q in %v", tok, got)
		}
	}
}

func TestDetectTokensFromComposerJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require": {"doctrine/dbal": "^3.0"}}`)

	got := DetectTokens(dir)
	if len(got) != 1 || got[0] != "postgres" {
		t.Fatalf("got %v", got)
	}
}

func TestDetectTokensNoSignals(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies": {"react": "^19.0.0"}}`)

	got := DetectTokens(dir)
	if len(got) != 0 {
		t.Fatalf("expected no tokens, got %v", got)
	}
}

func TestDetectTokensMissingFiles(t *testing.T) {
	dir := t.TempDir()
	got := DetectTokens(dir)
	if len(got) != 0 {
		t.Fatalf("expected no tokens, got %v", got)
	}
}
