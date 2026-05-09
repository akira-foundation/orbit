package proxy

import (
	"context"
	"sync"

	"orbit-app/internal/projects"
)

type Manager interface {
	RegisterDomain(ctx context.Context, p *projects.Project) error
	UnregisterDomain(ctx context.Context, p *projects.Project) error
	List(ctx context.Context) ([]string, error)
}

type stubManager struct {
	mu      sync.RWMutex
	domains map[string]string
}

func NewStub() Manager {
	return &stubManager{domains: make(map[string]string)}
}

func (m *stubManager) RegisterDomain(_ context.Context, p *projects.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.domains[p.LocalDomain] = p.ID
	return nil
}

func (m *stubManager) UnregisterDomain(_ context.Context, p *projects.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.domains, p.LocalDomain)
	return nil
}

func (m *stubManager) List(_ context.Context) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.domains))
	for d := range m.domains {
		out = append(out, d)
	}
	return out, nil
}
