package proxy

import (
	"context"
	"errors"
	"strings"

	"orbit-app/internal/projects"
)

var ErrDomainNotRegistered = errors.New("proxy: domain not registered with orbit")

type ProjectLookup interface {
	List(ctx context.Context) ([]projects.Project, error)
}

type Manager interface {
	Resolve(ctx context.Context, host string) (*projects.Project, error)
	ResolveByDevPort(ctx context.Context, port int) (*projects.Project, bool)
	List(ctx context.Context) ([]string, error)
}

type registry struct {
	projects     ProjectLookup
	domainSuffix string
}

func New(p ProjectLookup, domainSuffix string) Manager {
	return &registry{projects: p, domainSuffix: strings.TrimPrefix(domainSuffix, ".")}
}

func (r *registry) Resolve(ctx context.Context, host string) (*projects.Project, error) {
	host = normalizeHost(host)
	if host == "" {
		return nil, ErrDomainNotRegistered
	}

	suffix := "." + r.domainSuffix
	if !strings.HasSuffix(host, suffix) && host != r.domainSuffix {
		return nil, ErrDomainNotRegistered
	}

	all, err := r.projects.List(ctx)
	if err != nil {
		return nil, err
	}

	candidate := host
	for {
		for i := range all {
			if all[i].LocalDomain == candidate {
				return &all[i], nil
			}
		}
		idx := strings.IndexByte(candidate, '.')
		if idx < 0 {
			break
		}
		next := candidate[idx+1:]
		if next == r.domainSuffix || next == "" {
			break
		}
		candidate = next
	}

	return nil, ErrDomainNotRegistered
}

func (r *registry) ResolveByDevPort(ctx context.Context, port int) (*projects.Project, bool) {
	if port == 0 {
		return nil, false
	}
	all, err := r.projects.List(ctx)
	if err != nil {
		return nil, false
	}
	for i := range all {
		if all[i].DevPort == port {
			return &all[i], true
		}
	}
	return nil, false
}

func (r *registry) List(ctx context.Context) ([]string, error) {
	all, err := r.projects.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(all))
	for i := range all {
		out = append(out, all[i].LocalDomain)
	}
	return out, nil
}

func normalizeHost(host string) string {
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimSuffix(host, ".")
	return host
}
