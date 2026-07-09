package bindings

import (
	"context"

	"orbit-app/internal/parked"
)

type Parked struct {
	ctx     context.Context
	store   *parked.Store
	manager *parked.Manager
}

func NewParked() *Parked { return &Parked{} }

func (p *Parked) Attach(d Deps) {
	p.ctx = d.Ctx
	p.store = d.ParkedStore
	p.manager = d.ParkedManager
}

func (p *Parked) ParkedFolders() []string {
	return p.store.Folders()
}

func (p *Parked) AddParkedFolder(path string) error {
	if err := p.store.AddFolder(path); err != nil {
		return err
	}
	p.manager.Sync(p.ctx)
	return nil
}

func (p *Parked) RemoveParkedFolder(path string) error {
	return p.store.RemoveFolder(path)
}

func (p *Parked) ParkedProjectIDs() []string {
	return p.store.AutoAddedIDs()
}
