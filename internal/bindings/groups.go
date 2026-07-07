package bindings

import (
	"context"

	"orbit-app/internal/projects"
	"orbit-app/internal/runtime"
)

type Groups struct {
	ctx     context.Context
	service *projects.Service
	runtime runtime.Manager
}

func NewGroups() *Groups { return &Groups{} }

func (g *Groups) Attach(d Deps) {
	g.ctx = d.Ctx
	g.service = d.Service
	g.runtime = d.Runtime
}

func (g *Groups) ListGroups() ([]projects.Group, error) {
	return g.service.ListGroups(g.ctx)
}

func (g *Groups) CreateGroup(name string) (*projects.Group, error) {
	return g.service.CreateGroup(g.ctx, name)
}

func (g *Groups) DeleteGroup(id string) error {
	return g.service.DeleteGroup(g.ctx, id)
}

func (g *Groups) AddToGroup(groupID, projectID string) error {
	return g.service.AddToGroup(g.ctx, groupID, projectID)
}

func (g *Groups) RemoveFromGroup(groupID, projectID string) error {
	return g.service.RemoveFromGroup(g.ctx, groupID, projectID)
}

func (g *Groups) StartGroup(id string) error {
	members, err := g.service.GroupMembers(g.ctx, id)
	if err != nil {
		return err
	}
	for _, pid := range members {
		if err := g.runtime.Start(g.ctx, pid); err != nil {
			g.runtime.RecordEvent(pid, runtime.LevelError, runtime.SourceSystem,
				"group start failed: "+err.Error())
		}
	}
	return nil
}

func (g *Groups) StopGroup(id string) error {
	members, err := g.service.GroupMembers(g.ctx, id)
	if err != nil {
		return err
	}
	for _, pid := range members {
		_ = g.runtime.Stop(g.ctx, pid)
	}
	return nil
}
