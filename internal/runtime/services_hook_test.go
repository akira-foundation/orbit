package runtime

import (
	"context"
	"sync"
	"testing"
)

type fakeCoord struct {
	mu      sync.Mutex
	started []string
	stopped []string
	env     map[string]string
}

func (f *fakeCoord) OnProjectStart(_ context.Context, projectID, _, _ string) (map[string]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, projectID)
	return f.env, nil
}

func (f *fakeCoord) OnProjectStop(projectID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = append(f.stopped, projectID)
}

func TestManagerSetServicesStoresCoordinator(t *testing.T) {
	m := &manager{}
	c := &fakeCoord{}
	m.SetServices(c)
	if m.services == nil {
		t.Fatal("expected services coordinator set")
	}
	m.servicesOnStop("p1")
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.stopped) != 1 || c.stopped[0] != "p1" {
		t.Fatalf("stopped = %v", c.stopped)
	}
}

func TestServicesEnvInjectedOnlyWhenAbsent(t *testing.T) {
	base := []string{"HOME=/x", "DATABASE_URL=keepme"}
	got := mergeServiceEnv(base, map[string]string{
		"DATABASE_URL": "postgres://orbit@127.0.0.2:5432/app",
		"DB_HOST":      "127.0.0.2",
	})
	joined := map[string]bool{}
	for _, kv := range got {
		joined[kv] = true
	}
	if !joined["DATABASE_URL=keepme"] {
		t.Fatalf("existing var overwritten: %v", got)
	}
	if joined["DATABASE_URL=postgres://orbit@127.0.0.2:5432/app"] {
		t.Fatalf("injected var duplicated: %v", got)
	}
	if !joined["DB_HOST=127.0.0.2"] {
		t.Fatalf("absent var not injected: %v", got)
	}
}
