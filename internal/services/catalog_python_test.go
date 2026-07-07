package services

import (
	"strings"
	"testing"
)

func TestPythonVersionsPinned(t *testing.T) {
	versions := PythonVersions()
	if len(versions) != 3 {
		t.Fatalf("versions = %v", versions)
	}
	for _, v := range versions {
		e, ok := PythonEngineForVersion(v)
		if !ok {
			t.Fatalf("no engine for %s", v)
		}
		if e.Family != "python" || !e.ExtractTree {
			t.Fatalf("%s family=%q extractTree=%v", v, e.Family, e.ExtractTree)
		}
		if len(e.Platforms) != 4 {
			t.Fatalf("%s platforms = %d", v, len(e.Platforms))
		}
		for key, p := range e.Platforms {
			if len(p.SHA256) != 64 {
				t.Fatalf("%s/%s missing pinned sha256", v, key)
			}
			if p.ArchiveBinaryPath != "python/bin/python3" {
				t.Fatalf("%s/%s binary path = %q", v, key, p.ArchiveBinaryPath)
			}
			if !strings.Contains(p.URL, "install_only.tar.gz") {
				t.Fatalf("%s/%s url = %q", v, key, p.URL)
			}
		}
	}
}

func TestPythonEngineUnknownVersion(t *testing.T) {
	if _, ok := PythonEngineForVersion("2.7.0"); ok {
		t.Fatal("expected unknown version to miss")
	}
}
