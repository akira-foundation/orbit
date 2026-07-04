package proxy

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFastCGITransportRoutesDynamicRequest(t *testing.T) {
	bin := lookupPHPFPM(t)
	sockPath, docRoot := startTestFPM(t, bin)
	transport := &fcgiTransport{SockPath: sockPath, DocRoot: docRoot}

	req := httptest.NewRequest(http.MethodGet, "http://myapp.orbit.test/index.php?name=world", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if got := resp.Header.Get("X-Orbit-Test"); got != "1" {
		t.Fatalf("X-Orbit-Test header = %q", got)
	}
	body := readAll(t, resp.Body)
	if !strings.Contains(body, "hello world") {
		t.Fatalf("body = %q", body)
	}
}

func TestFastCGITransportHandlesNilRequestBody(t *testing.T) {
	bin := lookupPHPFPM(t)
	sockPath, docRoot := startTestFPM(t, bin)
	transport := &fcgiTransport{SockPath: sockPath, DocRoot: docRoot}

	req := httptest.NewRequest(http.MethodGet, "http://myapp.orbit.test/index.php", nil)
	req.Body = nil // httputil.ReverseProxy forwards a nil Body for bodyless requests
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestFastCGITransportServesStaticFileDirectly(t *testing.T) {
	bin := lookupPHPFPM(t)
	sockPath, docRoot := startTestFPM(t, bin)
	if err := os.WriteFile(filepath.Join(docRoot, "app.css"), []byte("body{color:red}"), 0o644); err != nil {
		t.Fatal(err)
	}
	transport := &fcgiTransport{SockPath: sockPath, DocRoot: docRoot}

	req := httptest.NewRequest(http.MethodGet, "http://myapp.orbit.test/app.css", nil)
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/css") {
		t.Fatalf("content-type = %q", ct)
	}
	body := readAll(t, resp.Body)
	if body != "body{color:red}" {
		t.Fatalf("body = %q", body)
	}
}

func TestResolveScriptTargetNeverServesPHPSourceAsStatic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.php"), []byte("<?php secret();"), 0o644); err != nil {
		t.Fatal(err)
	}
	script, static := resolveScriptTarget(dir, "/index.php")
	if static {
		t.Fatalf("a .php file must never be served as static source, got static=%v script=%q", static, script)
	}
}

func TestResolveScriptTargetBlocksPathTraversal(t *testing.T) {
	dir := t.TempDir()
	docRoot := filepath.Join(dir, "public")
	if err := os.MkdirAll(docRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(secret, []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	script, static := resolveScriptTarget(docRoot, "/../secret.txt")
	if static {
		t.Fatalf("path traversal should not resolve to a static file, got %q", script)
	}
	if script != filepath.Join(docRoot, "index.php") {
		t.Fatalf("expected fallback to index.php, got %q", script)
	}
}
