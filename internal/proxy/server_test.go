package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"orbit-app/internal/projects"
)

func TestServer_ProxiesHTTP(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Forwarded-Host"); got != "site.orbit.test" {
			t.Errorf("X-Forwarded-Host = %q, want site.orbit.test", got)
		}
		if got := r.Header.Get("X-Forwarded-Proto"); got != "http" {
			t.Errorf("X-Forwarded-Proto = %q, want http", got)
		}
		w.Header().Set("X-Hello", "world")
		_, _ = w.Write([]byte("hello from upstream"))
	}))
	defer upstream.Close()

	srv := newTestServer(t, upstream.URL)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Host = "site.orbit.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if resp.Header.Get("X-Hello") != "world" {
		t.Fatalf("upstream header lost")
	}
}

func TestServer_ProxiesWebSocket(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upstream upgrade: %v", err)
			return
		}
		defer c.Close()
		_, msg, err := c.ReadMessage()
		if err != nil {
			return
		}
		_ = c.WriteMessage(websocket.TextMessage, append([]byte("echo:"), msg...))
	}))
	defer upstream.Close()

	srv := newTestServer(t, upstream.URL)
	defer srv.Close()

	wsURL := "ws://" + strings.TrimPrefix(srv.URL, "http://")
	dialer := websocket.Dialer{HandshakeTimeout: 2 * time.Second}
	header := http.Header{"Host": []string{"site.orbit.test"}}
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(msg) != "echo:ping" {
		t.Fatalf("got %q, want echo:ping", msg)
	}
}

func TestServer_ProxiesFastCGI(t *testing.T) {
	bin := lookupPHPFPM(t)
	sockPath, docRoot := startTestFPM(t, bin)

	lk := &fakeLookup{items: []projects.Project{
		{
			ID: "1", Slug: "phpapp", LocalDomain: "phpapp.orbit.test",
			Path: strings.TrimSuffix(docRoot, "/public"), RuntimeKind: projects.RuntimeKindPHPFPM,
		},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{sockPath: sockPath, running: map[string]bool{"1": true}}
	router := NewRouter(reg, rt)
	router.health = HealthOptions{
		Timeout:      2 * time.Second,
		PollInterval: 50 * time.Millisecond,
		DialTimeout:  200 * time.Millisecond,
	}

	rec := NewRecoveryHandler(reg, rt)
	proxy := NewServer("", router, rec)
	ts := httptest.NewServer(proxy)
	defer ts.Close()
	defer proxy.Shutdown(context.Background())

	req, _ := http.NewRequest("GET", ts.URL+"/index.php?name=fastcgi", nil)
	req.Host = "phpapp.orbit.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body := readAll(t, resp.Body)
	if !strings.Contains(body, "hello fastcgi") {
		t.Fatalf("body = %q", body)
	}
}

func TestServer_NotRegistered(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstream.Close()

	srv := newTestServer(t, upstream.URL)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/", nil)
	req.Host = "ghost.orbit.test"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

// newTestServer wires up a Server in front of an httptest upstream for
// site.orbit.test, and starts it on a random local port.
func newTestServer(t *testing.T, upstreamURL string) *httptest.Server {
	t.Helper()

	_, portStr, _ := net.SplitHostPort(strings.TrimPrefix(upstreamURL, "http://"))
	port, _ := strconv.Atoi(portStr)

	lk := &fakeLookup{items: []projects.Project{
		{ID: "1", Slug: "site", LocalDomain: "site.orbit.test", DevPort: port},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{port: port, running: map[string]bool{"1": true}}
	router := NewRouter(reg, rt)
	router.health = HealthOptions{
		Timeout:      time.Second,
		PollInterval: 50 * time.Millisecond,
		DialTimeout:  200 * time.Millisecond,
	}

	rec := NewRecoveryHandler(reg, rt)
	proxy := NewServer("", router, rec)
	ts := httptest.NewServer(proxy)
	t.Cleanup(func() {
		_ = proxy.Shutdown(context.Background())
	})
	return ts
}
