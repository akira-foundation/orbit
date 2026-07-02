package services

import (
	"strings"
	"testing"
)

func TestCatalogHasMailpit(t *testing.T) {
	e, ok := ResolveEngine("mailpit")
	if !ok {
		t.Fatal("mailpit not in catalog")
	}
	if e.Version != "v1.30.3" {
		t.Fatalf("version = %q", e.Version)
	}
	if e.SMTPPort != 1025 || e.WebPort != 8025 || e.Port != 8025 {
		t.Fatalf("ports smtp=%d web=%d dial=%d", e.SMTPPort, e.WebPort, e.Port)
	}
	if e.Bind != "127.0.0.2" {
		t.Fatalf("bind = %q", e.Bind)
	}
	if e.WebDomain != "mail" {
		t.Fatalf("web domain = %q", e.WebDomain)
	}
}

func TestEngineArgsBindOnAliasIP(t *testing.T) {
	e, _ := ResolveEngine("mailpit")
	p := Platform{ArchiveBinaryPath: "mailpit"}
	args := strings.Join(e.Args(p, "/data/mailpit"), " ")
	if !strings.Contains(args, "127.0.0.2:1025") {
		t.Fatalf("smtp listen missing: %s", args)
	}
	if !strings.Contains(args, "127.0.0.2:8025") {
		t.Fatalf("web listen missing: %s", args)
	}
}

func TestResolveUnknownEngine(t *testing.T) {
	if _, ok := ResolveEngine("nope"); ok {
		t.Fatal("expected unknown engine to be absent")
	}
}
