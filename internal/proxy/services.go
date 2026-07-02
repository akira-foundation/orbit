package proxy

import (
	"net"
	"strings"
	"time"
)

type ServiceRoute struct {
	Engine      string
	DisplayName string
	Upstream    string
}

type ServiceResolver interface {
	ResolveService(host string) (ServiceRoute, bool)
}

type serviceTable struct {
	suffix  string
	entries map[string]ServiceRoute
}

func NewServiceTable(domainSuffix string, entries map[string]ServiceRoute) ServiceResolver {
	return &serviceTable{
		suffix:  strings.TrimPrefix(domainSuffix, "."),
		entries: entries,
	}
}

func (s *serviceTable) ResolveService(host string) (ServiceRoute, bool) {
	host = normalizeHost(host)
	sub := strings.TrimSuffix(host, "."+s.suffix)
	if sub == host {
		return ServiceRoute{}, false
	}
	route, ok := s.entries[sub]
	return route, ok
}

func dialUpstream(addr string) bool {
	conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
