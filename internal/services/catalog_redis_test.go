package services

import (
	"strings"
	"testing"
)

func TestRedisCatalogEntry(t *testing.T) {
	e, ok := ResolveEngine("redis")
	if !ok {
		t.Fatal("redis missing from catalog")
	}
	if e.Port != 40350 || e.Family != "redis" {
		t.Fatalf("port = %d family %q", e.Port, e.Family)
	}
	if !e.Zip || e.RawBinary || e.ExtractTree {
		t.Fatalf("expected zip mode")
	}
	if e.Provision == nil || e.Init != nil {
		t.Fatalf("expected Provision hook and no Init")
	}
	for key, p := range e.Platforms {
		if len(p.SHA256) != 64 {
			t.Fatalf("%s missing pinned sha256", key)
		}
		if p.ArchiveBinaryPath != "redka" {
			t.Fatalf("%s binary path = %q", key, p.ArchiveBinaryPath)
		}
	}
}

func TestRedisArgsBindAndPort(t *testing.T) {
	e, _ := ResolveEngine("redis")
	args := strings.Join(e.Args(Platform{}, "/data/redis"), " ")
	for _, want := range []string{"-h 127.0.0.2", "-p 40350", "/data/redis/redka.db"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args missing %q: %s", want, args)
		}
	}
}

func TestRedisEnvShape(t *testing.T) {
	env := RedisEnv("127.0.0.2", 40350)
	if env["REDIS_URL"] != "redis://127.0.0.2:40350" || env["REDIS_HOST"] != "127.0.0.2" {
		t.Fatalf("env = %v", env)
	}
	if env["REDIS_PORT"] != "40350" {
		t.Fatalf("env = %v", env)
	}
}
