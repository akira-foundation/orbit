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
}

func (f *fakeCoord) OnProjectStart(_ context.Context, projectID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = append(f.started, projectID)
	return nil
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
