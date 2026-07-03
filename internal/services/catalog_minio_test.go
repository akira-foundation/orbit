package services

import (
	"strings"
	"testing"
)

func TestMinioCatalogEntry(t *testing.T) {
	e, ok := ResolveEngine("minio")
	if !ok {
		t.Fatal("minio missing from catalog")
	}
	if e.Port != 40340 || e.WebPort != 40341 || e.WebDomain != "s3" {
		t.Fatalf("ports = %d/%d domain %q", e.Port, e.WebPort, e.WebDomain)
	}
	if !e.RawBinary || e.ExtractTree {
		t.Fatalf("expected raw binary mode")
	}
	if e.Provision == nil || e.Init != nil {
		t.Fatalf("expected Provision hook and no Init")
	}
	if len(e.Env) != 2 {
		t.Fatalf("env = %v", e.Env)
	}
	if len(e.Platforms) != 4 {
		t.Fatalf("platforms = %d", len(e.Platforms))
	}
	for key, p := range e.Platforms {
		if len(p.SHA256) != 64 {
			t.Fatalf("%s missing pinned sha256", key)
		}
	}
}

func TestMinioArgsBindAliasIP(t *testing.T) {
	e, _ := ResolveEngine("minio")
	args := strings.Join(e.Args(Platform{}, "/data/minio"), " ")
	for _, want := range []string{"server /data/minio", "--address 127.0.0.2:40340", "--console-address 127.0.0.2:40341"} {
		if !strings.Contains(args, want) {
			t.Fatalf("args missing %q: %s", want, args)
		}
	}
}

func TestMinioEnvShape(t *testing.T) {
	env := MinioEnv("127.0.0.2", 40340, "shop")
	if env["AWS_BUCKET"] != "shop" || env["AWS_ENDPOINT"] != "http://127.0.0.2:40340" {
		t.Fatalf("env = %v", env)
	}
	if env["AWS_USE_PATH_STYLE_ENDPOINT"] != "true" || env["AWS_ACCESS_KEY_ID"] != "orbit" {
		t.Fatalf("env = %v", env)
	}
}
