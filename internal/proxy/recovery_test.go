package proxy

import (
	"bufio"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"orbit-app/internal/projects"
)

func newProxyTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestRecovery_OverlayOnUpstreamFailure(t *testing.T) {
	lk := &fakeLookup{items: []projects.Project{
		{ID: "1", Name: "site", Slug: "site", LocalDomain: "site.orbit.test", DevPort: 1},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{port: 1, running: map[string]bool{"1": true}}
	router := NewRouter(reg, rt)
	router.health = HealthOptions{Timeout: 200 * time.Millisecond, PollInterval: 50 * time.Millisecond, DialTimeout: 100 * time.Millisecond}
	rec := NewRecoveryHandler(reg, rt)
	srv := newProxyTestServer(t, NewServer("", router, rec))

	req, _ := http.NewRequest("GET", srv.URL+"/about", nil)
	req.Host = "site.orbit.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", resp.StatusCode)
	}
	body := readAll(t, resp.Body)
	if !strings.Contains(body, "Reconnecting site") {
		t.Fatalf("expected recovery overlay, body=%s", body[:min(len(body), 200)])
	}
	if !strings.Contains(body, "/__orbit__/recovery/events") {
		t.Fatal("expected SSE endpoint reference in overlay")
	}
}

func TestRecovery_StatusEvents(t *testing.T) {
	lk := &fakeLookup{items: []projects.Project{
		{ID: "1", Slug: "site", LocalDomain: "site.orbit.test"},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{}
	router := NewRouter(reg, rt)
	rec := NewRecoveryHandler(reg, rt)
	srv := newProxyTestServer(t, NewServer("", router, rec))

	req, _ := http.NewRequest("GET", srv.URL+"/__orbit__/recovery/events", nil)
	req.Host = "site.orbit.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q", ct)
	}
	scanner := bufio.NewScanner(resp.Body)
	got := false
	timeout := time.After(2 * time.Second)
	done := make(chan bool, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data: ") {
				got = true
				done <- true
				return
			}
		}
		done <- false
	}()
	select {
	case <-done:
	case <-timeout:
	}
	if !got {
		t.Fatal("expected at least one SSE data frame")
	}
}

func TestRecovery_RestartEndpoint(t *testing.T) {
	lk := &fakeLookup{items: []projects.Project{
		{ID: "1", Slug: "site", LocalDomain: "site.orbit.test"},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{}
	router := NewRouter(reg, rt)
	rec := NewRecoveryHandler(reg, rt)
	srv := newProxyTestServer(t, NewServer("", router, rec))

	req, _ := http.NewRequest("POST", srv.URL+"/__orbit__/recovery/restart", nil)
	req.Host = "site.orbit.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
	if !rt.IsRunning("1") {
		t.Fatal("expected runtime started after restart endpoint")
	}
}
