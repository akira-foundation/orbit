package bindings

import "orbit-app/internal/proxy"

type Requests struct {
	proxy *proxy.Server
}

func NewRequests() *Requests { return &Requests{} }

func (r *Requests) Attach(d Deps) {
	r.proxy = d.ProxyServer
}

func (r *Requests) ProjectRequests(projectID string) []proxy.RequestEntry {
	if r.proxy == nil {
		return []proxy.RequestEntry{}
	}
	return r.proxy.RecentRequests(projectID)
}
