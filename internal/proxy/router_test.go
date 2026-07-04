package proxy

import (
	"context"
	"net"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"orbit-app/internal/projects"
	"orbit-app/internal/runtime"
)

type fakeRuntime struct {
	mu       sync.Mutex
	running  map[string]bool
	port     int
	sockPath string
	logs     []runtime.LogLine
}

func (f *fakeRuntime) Start(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.running == nil {
		f.running = map[string]bool{}
	}
	f.running[id] = true
	return nil
}

func (f *fakeRuntime) Restart(ctx context.Context, id string) error {
	return f.Start(ctx, id)
}

func (f *fakeRuntime) Port(string) int { return f.port }

func (f *fakeRuntime) SocketPath(string) string { return f.sockPath }

func (f *fakeRuntime) ConnOpen(string)                                        {}
func (f *fakeRuntime) ConnClose(string)                                       {}
func (f *fakeRuntime) RecordRequest(string, int, float64, int64, int64, bool) {}

func (f *fakeRuntime) IsRunning(id string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.running[id]
}

func (f *fakeRuntime) Status(id string) runtime.Snapshot {
	st := projects.StatusStopped
	if f.IsRunning(id) {
		st = projects.StatusRunning
	}
	return runtime.Snapshot{ProjectID: id, Status: st, Port: f.port}
}

func (f *fakeRuntime) Logs(string) []runtime.LogLine { return f.logs }

func TestRouter_Route(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()
	_, portStr, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	port, _ := strconv.Atoi(portStr)

	lk := &fakeLookup{items: []projects.Project{
		{ID: "1", Slug: "my-app", LocalDomain: "my-app.orbit.test", DevPort: port},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{port: port}
	router := NewRouter(reg, rt)
	router.health = HealthOptions{
		Timeout:      time.Second,
		PollInterval: 50 * time.Millisecond,
		DialTimeout:  200 * time.Millisecond,
	}

	target, err := router.Route(context.Background(), "my-app.orbit.test")
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if target.Project.ID != "1" {
		t.Fatalf("unexpected project: %s", target.Project.ID)
	}
	if !rt.IsRunning("1") {
		t.Fatalf("expected runtime to be started")
	}
	if !strings.Contains(target.URL.String(), strconv.Itoa(port)) {
		t.Fatalf("target URL missing port: %s", target.URL)
	}
}

func TestRouter_RoutePHPFPM(t *testing.T) {
	lk := &fakeLookup{items: []projects.Project{
		{
			ID: "1", Slug: "my-app", LocalDomain: "my-app.orbit.test",
			Path: "/srv/my-app", RuntimeKind: projects.RuntimeKindPHPFPM,
		},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{sockPath: "/tmp/orbit-my-app.sock"}
	router := NewRouter(reg, rt)
	router.health = HealthOptions{
		Timeout:      time.Second,
		PollInterval: 50 * time.Millisecond,
		DialTimeout:  200 * time.Millisecond,
	}

	target, err := router.Route(context.Background(), "my-app.orbit.test")
	if err != nil {
		t.Fatalf("Route: %v", err)
	}
	if target.Kind != TargetFastCGI {
		t.Fatalf("Kind = %q, want fastcgi", target.Kind)
	}
	if target.SockPath != "/tmp/orbit-my-app.sock" {
		t.Fatalf("SockPath = %q", target.SockPath)
	}
	if target.DocRoot != "/srv/my-app/public" {
		t.Fatalf("DocRoot = %q, want /srv/my-app/public", target.DocRoot)
	}
	if target.URL != nil {
		t.Fatalf("expected no URL for a fastcgi target, got %v", target.URL)
	}
}

func TestRouter_RoutePHPFPMTimesOutWithoutSocket(t *testing.T) {
	lk := &fakeLookup{items: []projects.Project{
		{ID: "1", Slug: "my-app", LocalDomain: "my-app.orbit.test", RuntimeKind: projects.RuntimeKindPHPFPM},
	}}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{}
	router := NewRouter(reg, rt)
	router.health = HealthOptions{
		Timeout:      100 * time.Millisecond,
		PollInterval: 20 * time.Millisecond,
		DialTimeout:  50 * time.Millisecond,
	}

	_, err := router.Route(context.Background(), "my-app.orbit.test")
	if err == nil {
		t.Fatal("expected error when php-fpm socket never becomes available")
	}
}

func TestRouter_NotRegistered(t *testing.T) {
	lk := &fakeLookup{}
	reg := New(lk, "orbit.test")
	rt := &fakeRuntime{}
	router := NewRouter(reg, rt)
	_, err := router.Route(context.Background(), "ghost.orbit.test")
	if err == nil {
		t.Fatal("expected error")
	}
}
