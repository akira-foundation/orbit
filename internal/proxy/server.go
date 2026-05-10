package proxy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

type Server struct {
	addr   string
	router *Router
	http   *http.Server
}

func NewServer(addr string, router *Router) *Server {
	s := &Server{addr: addr, router: router}
	s.http = &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
	}
	return s
}

func (s *Server) ListenAndServe() error {
	log.Printf("[proxy] listening on %s", s.addr)
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	target, err := s.router.Route(r.Context(), r.Host)
	if err != nil {
		s.handleError(w, r, err)
		return
	}

	rp := httputil.NewSingleHostReverseProxy(target.URL)
	rp.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, perr error) {
		log.Printf("[proxy] upstream error host=%s target=%s err=%v", r.Host, target.URL, perr)
		writeBadGateway(rw, target.URL.String(), perr)
	}

	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.URL.Host
		req.Header.Set("X-Forwarded-Host", r.Host)
		req.Header.Set("X-Forwarded-Proto", schemeOf(r))
		if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			req.Header.Set("X-Forwarded-For", ip)
		}
	}

	rp.ServeHTTP(w, r)
}

func (s *Server) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrDomainNotRegistered) {
		writeNotRegistered(w, r.Host)
		return
	}
	if errors.Is(err, ErrUnhealthy) {
		log.Printf("[proxy] unhealthy host=%s err=%v", r.Host, err)
		writeUnhealthy(w, r.Host)
		return
	}
	if errors.Is(err, ErrNoPort) {
		writeNoPort(w, r.Host)
		return
	}
	log.Printf("[proxy] route error host=%s err=%v", r.Host, err)
	http.Error(w, "Orbit: routing error", http.StatusInternalServerError)
}

func schemeOf(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if v := r.Header.Get("X-Forwarded-Proto"); v != "" {
		return v
	}
	return "http"
}

func writeNotRegistered(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, errorPage(
		"Domain not registered",
		fmt.Sprintf("%s is not registered with Orbit.", strings.TrimSpace(host)),
		"Add the project in Orbit to enable this domain.",
	))
}

func writeUnhealthy(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprint(w, errorPage(
		"Runtime not responding",
		fmt.Sprintf("Started the runtime for %s but it didn't become reachable.", host),
		"Check the runtime logs in Orbit for errors.",
	))
}

func writeNoPort(w http.ResponseWriter, host string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprint(w, errorPage(
		"No detected port",
		fmt.Sprintf("Could not determine which port the runtime for %s is listening on.", host),
		"Make sure your dev command prints the port (e.g. http://localhost:3000) on startup.",
	))
}

func writeBadGateway(w http.ResponseWriter, target string, err error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprint(w, errorPage(
		"Upstream error",
		fmt.Sprintf("Failed to reach %s.", target),
		err.Error(),
	))
}

func errorPage(title, lead, detail string) string {
	return fmt.Sprintf(`<!doctype html>
<html><head><meta charset="utf-8"><title>Orbit · %[1]s</title>
<style>
  :root { color-scheme: dark; }
  body { font-family: -apple-system, BlinkMacSystemFont, "SF Pro Text", system-ui, sans-serif;
         background: #0c0d12; color: #e5e7eb; margin: 0;
         min-height: 100vh; display: flex; align-items: center; justify-content: center; }
  .card { max-width: 520px; padding: 32px 36px; border: 1px solid rgba(255,255,255,0.08);
          border-radius: 16px; background: rgba(255,255,255,0.02); }
  h1 { font-size: 18px; margin: 0 0 12px; font-weight: 600; }
  p  { font-size: 14px; line-height: 1.5; color: #a1a1aa; margin: 0 0 8px; }
  .badge { display: inline-block; padding: 2px 8px; border-radius: 999px;
           background: rgba(124,127,255,0.12); color: #a5b4fc; font-size: 11px;
           letter-spacing: 0.08em; text-transform: uppercase; margin-bottom: 14px; }
</style></head>
<body><div class="card"><div class="badge">Orbit</div><h1>%[1]s</h1><p>%[2]s</p><p>%[3]s</p></div></body></html>`,
		title, lead, detail)
}
