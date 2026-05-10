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

	host, herr := dialableHost(ctx, port, r.health)
	if herr != nil {
		return nil, herr
	}

	u, perr := url.Parse(fmt.Sprintf("http://%s:%d", host, port))
	if perr != nil {
		return nil, perr
	}
	return &Target{Project: proj, URL: u}, nil
}

func (r *Router) waitForPort(ctx context.Context, proj *projects.Project) (int, error) {
	// Always wait for the runtime to advertise the port the framework
	// actually bound. Don't fall back to the analyzed devPort — frameworks
	// often pick a different port (Astro auto-increments on conflict, Vite
	// reads vite.config, Next obeys $PORT) and routing to the wrong one
	// just yields a useless 502.
	deadline := time.Now().Add(r.health.Timeout)
	for {
		if p := r.runtime.Port(proj.ID); p > 0 {
			return p, nil
		}
		if time.Now().After(deadline) {
			return 0, ErrNoPort
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(r.health.PollInterval):
		}
	}
}
