package services

import (
	"fmt"
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
	if e.SMTPPort != 40322 || e.WebPort != 40321 || e.Port != 40321 {
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
	if !strings.Contains(args, "127.0.0.2:40322") {
		t.Fatalf("smtp listen missing: %s", args)
	}
	if !strings.Contains(args, "127.0.0.2:40321") {
		t.Fatalf("web listen missing: %s", args)
	}
}

func TestResolveUnknownEngine(t *testing.T) {
	if _, ok := ResolveEngine("nope"); ok {
		t.Fatal("expected unknown engine to be absent")
	}
}

func TestMailpitSetupAdvertisesActualSMTPPort(t *testing.T) {
	e, _ := ResolveEngine("mailpit")
	port := fmt.Sprintf("%d", e.SMTPPort)

	var portField string
	for _, f := range e.Setup.Fields {
		if f.Label == "SMTP port" {
			portField = f.Value
		}
	}
	if portField != port {
		t.Fatalf("setup SMTP port field = %q, want %q", portField, port)
	}

	for _, s := range e.Setup.Snippets {
		if strings.Contains(s.Code, "1025") || !strings.Contains(s.Code, port) {
			t.Fatalf("snippet %q does not advertise port %s: %s", s.Label, port, s.Code)
		}
	}
}
