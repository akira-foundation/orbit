package proxy

import (
	"errors"
	"fmt"
	"net/http"
)

type Handler struct {
	mgr Manager
}

func NewHandler(m Manager) *Handler {
	return &Handler{mgr: m}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	proj, err := h.mgr.Resolve(r.Context(), r.Host)
	if err != nil {
		if errors.Is(err, ErrDomainNotRegistered) {
			h.notRegistered(w, r.Host)
			return
		}
		http.Error(w, "proxy: resolve error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotImplemented)
	fmt.Fprintf(w, "Orbit: would proxy %s to %s (not yet implemented)\n", r.Host, proj.LocalDomain)
}

func (h *Handler) notRegistered(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "Orbit: %q is not registered. Add the project in Orbit to enable this domain.\n", host)
}
