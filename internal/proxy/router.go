package proxy

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"orbit-app/internal/projects"
)

var ErrNoPort = errors.New("proxy: project has no known port")

type RuntimeProvider interface {
	Start(ctx context.Context, projectID string) error
	Port(projectID string) int
	IsRunning(projectID string) bool
}

type Target struct {
	Project *projects.Project
	URL     *url.URL
}

type Router struct {
	registry Manager
	runtime  RuntimeProvider
	health   HealthOptions
}

func NewRouter(reg Manager, rt RuntimeProvider) *Router {
	return &Router{registry: reg, runtime: rt, health: defaultHealthOptions()}
}

func (r *Router) Route(ctx context.Context, host string) (*Target, error) {
	proj, err := r.registry.Resolve(ctx, host)
	if err != nil {
		return nil, err
	}

	if !r.runtime.IsRunning(proj.ID) {
		if err := r.runtime.Start(ctx, proj.ID); err != nil {
			return nil, fmt.Errorf("proxy: wake runtime: %w", err)
		}
	}

	port, err := r.waitForPort(ctx, proj)
	if err != nil {
		return nil, err
	}

	if err := WaitTCP(ctx, "127.0.0.1", port, r.health); err != nil {
		return nil, err
	}

	u, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}
	return &Target{Project: proj, URL: u}, nil
}

func (r *Router) waitForPort(ctx context.Context, proj *projects.Project) (int, error) {
	if p := r.runtime.Port(proj.ID); p > 0 {
		return p, nil
	}
	if proj.DevPort > 0 {
		return proj.DevPort, nil
	}

	deadline := time.Now().Add(r.health.Timeout)
	for time.Now().Before(deadline) {
		if p := r.runtime.Port(proj.ID); p > 0 {
			return p, nil
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(r.health.PollInterval):
		}
	}
	return 0, ErrNoPort
}
