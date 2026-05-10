package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

type Server struct {
	addr      string
	router    *Router
	http      *http.Server
	transport *http.Transport
}

func NewServer(addr string, router *Router) *Server {
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       120 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		// Keep upstream HTTP/1.1 so websocket Upgrade works reliably with all
		// JS dev servers. ResponseHeaderTimeout is left zero so long-polling
		// and slow first-render dev builds don't get killed mid-flight.
		ForceAttemptHTTP2: false,
	}

	s := &Server{addr: addr, router: router, transport: tr}
	s.http = &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		// ReadTimeout / WriteTimeout / IdleTimeout intentionally unset:
		// HMR websockets and SSE streams must stay open indefinitely.
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

	rp := &httputil.ReverseProxy{
		Transport:     s.transport,
		FlushInterval: -1, // flush after every write -> SSE / streamed responses
		Director: func(req *http.Request) {
			req.URL.Scheme = target.URL.Scheme
			req.URL.Host = target.URL.Host
			req.Host = target.URL.Host
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}
			req.Header.Set("X-Forwarded-Host", r.Host)
			req.Header.Set("X-Forwarded-Proto", schemeOf(r))
			if ip, _, e := net.SplitHostPort(r.RemoteAddr); e == nil {
				req.Header.Set("X-Forwarded-For", ip)
			}
		},
		ErrorHandler: func(rw http.ResponseWriter, req *http.Request, perr error) {
			if isClientGone(perr) || isWebSocket(req) {
				return
			}
			log.Printf("[proxy] upstream error host=%s target=%s err=%v", r.Host, target.URL, perr)
			writeBadGateway(rw, target.URL.String(), perr)
		},
	}

	rp.ServeHTTP(w, r)
}

func isWebSocket(r *http.Request) bool {
	if r == nil {
		return false
	}
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func isClientGone(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, io.EOF) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "use of closed network connection")
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
