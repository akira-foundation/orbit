package runtime

import (
	"context"
	"errors"
	"sync"

	"orbit-app/internal/projects"
)

var ErrNotImplemented = errors.New("runtime: not implemented in v1")

type Manager interface {
	Start(ctx context.Context, projectID string) error
	Stop(ctx context.Context, projectID string) error
	Restart(ctx context.Context, projectID string) error
	GetStatus(ctx context.Context, projectID string) (projects.Status, error)
}

type stubManager struct {
	mu    sync.RWMutex
	state map[string]projects.Status
}

func NewStub() Manager {
	return &stubManager{state: make(map[string]projects.Status)}
}

func (m *stubManager) Start(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state[id] = projects.StatusRunning
	return nil
}

func (m *stubManager) Stop(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state[id] = projects.StatusStopped
	return nil
}

func (m *stubManager) Restart(ctx context.Context, id string) error {
	if err := m.Stop(ctx, id); err != nil {
		return err
	}
	return m.Start(ctx, id)
}

func (m *stubManager) GetStatus(_ context.Context, id string) (projects.Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if s, ok := m.state[id]; ok {
		return s, nil
	}
	return projects.StatusStopped, nil
}
