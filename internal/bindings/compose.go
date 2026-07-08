package bindings

import (
	"context"
	"time"

	"orbit-app/internal/compose"
	"orbit-app/internal/projects"
)

type Compose struct {
	ctx     context.Context
	service *projects.Service
}

func NewCompose() *Compose { return &Compose{} }

func (c *Compose) Attach(d Deps) {
	c.ctx = d.Ctx
	c.service = d.Service
}

type ComposeInfo struct {
	Detected  bool              `json:"detected"`
	File      string            `json:"file"`
	Available bool              `json:"available"`
	Services  []compose.Service `json:"services"`
}

func (c *Compose) ComposeInfo(projectID string) (ComposeInfo, error) {
	proj, err := c.service.Get(c.ctx, projectID)
	if err != nil {
		return ComposeInfo{}, err
	}
	file, ok := compose.Detect(proj.Path)
	if !ok {
		return ComposeInfo{}, nil
	}
	info := ComposeInfo{Detected: true, File: file, Available: compose.Available()}
	if !info.Available {
		return info, nil
	}
	ctx, cancel := context.WithTimeout(c.ctx, 15*time.Second)
	defer cancel()
	if services, err := compose.Ps(ctx, proj.Path, file); err == nil {
		info.Services = services
	}
	return info, nil
}
