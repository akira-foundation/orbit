package parked

import (
	"context"
	"log"
	"os"
	"time"

	"orbit-app/internal/projects"
)

const syncInterval = 30 * time.Second

type Emitter func(name string, data ...any)

type Manager struct {
	store   *Store
	service *projects.Service
	emit    Emitter
}

func NewManager(store *Store, service *projects.Service, emit Emitter) *Manager {
	if emit == nil {
		emit = func(string, ...any) {}
	}
	return &Manager{store: store, service: service, emit: emit}
}

func (m *Manager) Start(ctx context.Context) {
	m.Sync(ctx)
	tk := time.NewTicker(syncInterval)
	defer tk.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			m.Sync(ctx)
		}
	}
}

func (m *Manager) Sync(ctx context.Context) {
	m.removeMissing(ctx)
	m.addDiscovered(ctx)
}

func (m *Manager) addDiscovered(ctx context.Context) {
	existing := map[string]bool{}
	if list, err := m.service.List(ctx); err == nil {
		for _, p := range list {
			existing[p.Path] = true
		}
	}
	for _, folder := range m.store.Folders() {
		for _, path := range DiscoverProjects(folder) {
			if existing[path] {
				continue
			}
			proj, err := m.service.Create(ctx, path)
			if err != nil {
				log.Printf("[parked] skip %s: %v", path, err)
				continue
			}
			existing[path] = true
			_ = m.store.MarkRegistered(path, proj.ID)
			log.Printf("[parked] registered %s (%s)", proj.Name, path)
			m.emit("parked:project-added", proj.Name)
		}
	}
}

func (m *Manager) removeMissing(ctx context.Context) {
	for path, id := range m.store.Registered() {
		if _, err := os.Stat(path); err == nil {
			continue
		}
		proj, err := m.service.Get(ctx, id)
		if err != nil {
			_ = m.store.Unregister(path)
			continue
		}
		if err := m.service.Delete(ctx, id); err != nil {
			log.Printf("[parked] remove %s: %v", path, err)
			continue
		}
		_ = m.store.Unregister(path)
		log.Printf("[parked] unregistered %s (folder gone)", proj.Name)
		m.emit("parked:project-removed", proj.Name)
	}
}
