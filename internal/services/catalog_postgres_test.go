package services

import (
	"strings"
	"testing"
)

func TestPostgresCatalogEntries(t *testing.T) {
	wantPorts := map[string]int{
		"postgres-18": 40330,
		"postgres-17": 40331,
		"postgres-16": 40332,
		"postgres-15": 40333,
	}
	for id, port := range wantPorts {
		e, ok := ResolveEngine(id)
		if !ok {
			t.Fatalf("%s missing from catalog", id)
		}
		if e.Port != port || e.Family != "postgres" || !e.ExtractTree {
			t.Fatalf("%s = port %d family %q tree %v", id, e.Port, e.Family, e.ExtractTree)
		}
		if e.WebPort != 0 {
			t.Fatalf("%s should have no web console", id)
		}
		if e.Init == nil || e.Provision == nil {
			t.Fatalf("%s missing Init/Provision hooks", id)
		}
		if len(e.Platforms) != 4 {
			t.Fatalf("%s platforms = %d", id, len(e.Platforms))
		}
		for key, p := range e.Platforms {
			if len(p.SHA256) != 64 {
				t.Fatalf("%s %s missing pinned sha256", id, key)
			}
			if !strings.Contains(p.ArchiveBinaryPath, "/bin/postgres") {
				t.Fatalf("%s %s bad binary path %q", id, key, p.ArchiveBinaryPath)
			}
		}
	}
}

func TestPostgresArgsBindAliasIPNoUnixSockets(t *testing.T) {
	e, _ := ResolveEngine("postgres-18")
	p := e.Platforms["darwin/arm64"]
	args := strings.Join(e.Args(p, "/data/postgres-18"), " ")
	for _, want := range []string{"-D /data/postgres-18", "-p 40330", "listen_addresses=127.0.0.2", "unix_socket_directories="} {
		if !strings.Contains(args, want) {
			t.Fatalf("args missing %q: %s", want, args)
		}
	}
}
