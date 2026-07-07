package proxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"orbit-app/internal/projects"
)

type Server struct {
	addr           string
	tlsAddr        string
	tlsCert        string
	tlsKey         string
	router         *Router
	recovery       *RecoveryHandler
	services       ServiceResolver
	serviceHandler *ServiceHandler
	http           *http.Server
	https          *http.Server
	lan            *http.Server
	suffix         string
	transport      *http.Transport
}

type Options struct {
	Addr           string
	TLSAddr        string
	TLSCert        string
	TLSKey         string
	Suffix         string
	Router         *Router
	Recovery       *RecoveryHandler
	Services       ServiceResolver
	ServiceStarter ServiceStarter
}

func NewServer(addr string, router *Router, recovery *RecoveryHandler) *Server {
	return NewServerWithOptions(Options{Addr: addr, Router: router, Recovery: recovery})
}

func NewServerWithOptions(opts Options) *Server {
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
		ForceAttemptHTTP2:     false,
	}

	s := &Server{
		addr:      opts.Addr,
		tlsAddr:   opts.TLSAddr,
		tlsCert:   opts.TLSCert,
		tlsKey:    opts.TLSKey,
		router:    opts.Router,
		recovery:  opts.Recovery,
		services:  opts.Services,
		suffix:    opts.Suffix,
		transport: tr,
	}
	if opts.Services != nil && opts.ServiceStarter != nil {
		s.serviceHandler = NewServiceHandler(opts.Services, opts.ServiceStarter)
	}
	s.http = &http.Server{
		Addr:              opts.Addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if opts.TLSAddr != "" && opts.TLSCert != "" && opts.TLSKey != "" {
		cache := newCertCache(opts.TLSCert, opts.TLSKey)
		s.https = &http.Server{
			Addr:              opts.TLSAddr,
			Handler:           s,
			ReadHeaderTimeout: 10 * time.Second,
			TLSConfig:         &tls.Config{GetCertificate: cache.GetCertificate},
		}
	}
	return s
}

func (s *Server) ListenAndServe() error {
	if s.https != nil {
		go func() {
			log.Printf("[proxy] tls listening on %s", s.tlsAddr)
			if err := s.https.ListenAndServeTLS("", ""); err != nil &&
				!errors.Is(err, http.ErrServerClosed) {
				log.Printf("[proxy] tls server error: %v", err)
			}
		}()
	}
	log.Printf("[proxy] listening on %s", s.addr)
	err := s.http.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.lan != nil {
		_ = s.lan.Shutdown(ctx)
	}
	if s.https != nil {
		_ = s.https.Shutdown(ctx)
	}
	if s.http == nil {
		return nil
	}
	return s.http.Shutdown(ctx)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rewriteShareHost(r, s.suffix)

	if s.serviceHandler != nil && strings.HasPrefix(r.URL.Path, servicePrefix+"/") {
		s.serviceHandler.ServeHTTP(w, r)
		return
	}

	if s.services != nil {
		if route, ok := s.services.ResolveService(r.Host); ok {
			if isPageNavigation(r) && !dialUpstream(route.Upstream) {
				s.renderServiceWake(w, r, route)
				return
			}
			s.proxyToService(w, r, route.Upstream)
			return
		}
	}

	if s.recovery != nil && strings.HasPrefix(r.URL.Path, "/__orbit__/") {
		s.recovery.ServeHTTP(w, r)
		return
	}

	if proj, ok := s.lookup(r); ok {
		w.Header().Set("Strict-Transport-Security", "max-age=0")
		if proj.Secure && r.TLS == nil {
			u := *r.URL
			u.Scheme = "https"
			u.Host = r.Host
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}
		if !proj.Secure && r.TLS != nil {
			u := *r.URL
			u.Scheme = "http"
			u.Host = r.Host
			http.Redirect(w, r, u.String(), http.StatusFound)
			return
		}
	}

	if s.recovery != nil && isPageNavigation(r) {
		if proj, ok := s.lookup(r); ok {
			st := s.router.runtime.Status(proj.ID)
			needsWake := st.Status == projects.StatusStopped ||
				st.Status == projects.StatusStarting
			if needsWake {
				if st.Status == projects.StatusStopped {
					go func() {
						ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
						defer cancel()
						_ = s.router.runtime.Start(ctx, proj.ID)
					}()
				}
				s.renderWake(w, r, proj)
				return
			}
		}
	}

	target, err := s.router.Route(r.Context(), r.Host)
	if err != nil {
		s.handleError(w, r, err)
		return
	}

	s.router.runtime.ConnOpen(target.Project.ID)
	defer s.router.runtime.ConnClose(target.Project.ID)

	start := time.Now()
	rec := &meteredWriter{ResponseWriter: w, status: http.StatusOK}
	bytesIn := r.ContentLength
	if bytesIn < 0 {
		bytesIn = 0
	}
	isWS := isWebSocket(r)
	defer func() {
		ms := float64(time.Since(start).Microseconds()) / 1000.0
		s.router.runtime.RecordRequest(
			target.Project.ID, rec.status, ms, bytesIn, rec.bytes, isWS,
		)
	}()

	targetDesc := target.SockPath
	if target.URL != nil {
		targetDesc = target.URL.String()
	}

	rp := &httputil.ReverseProxy{
		Transport:     s.transport,
		FlushInterval: -1, // flush after every write -> SSE / streamed responses
		Director: func(req *http.Request) {
			if target.URL != nil {
				req.URL.Scheme = target.URL.Scheme
				req.URL.Host = target.URL.Host
				_, port, _ := net.SplitHostPort(target.URL.Host)
				if port == "" {
					req.Host = "localhost"
				} else {
					req.Host = "localhost:" + port
				}
			}
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
			log.Printf("[proxy] upstream error host=%s target=%s err=%v", r.Host, targetDesc, perr)
			s.renderRecovery(rw, r)
		},
	}
	if target.Kind == TargetFastCGI {
		rp.Transport = &fcgiTransport{SockPath: target.SockPath, DocRoot: target.DocRoot}
	}

	rp.ServeHTTP(rec, r)
}

type meteredWriter struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
}

func (m *meteredWriter) WriteHeader(code int) {
	if !m.wroteHeader {
		m.status = code
		m.wroteHeader = true
	}
	m.ResponseWriter.WriteHeader(code)
}

func (m *meteredWriter) Write(b []byte) (int, error) {
	if !m.wroteHeader {
		m.status = http.StatusOK
		m.wroteHeader = true
	}
	n, err := m.ResponseWriter.Write(b)
	m.bytes += int64(n)
	return n, err
}

func (m *meteredWriter) Flush() {
	if f, ok := m.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (m *meteredWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := m.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

func (s *Server) renderServiceWake(w http.ResponseWriter, r *http.Request, route ServiceRoute) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprint(w, serviceWakePage(route, normalizeHost(r.Host)))
}

func (s *Server) proxyToService(w http.ResponseWriter, r *http.Request, upstream string) {
	target := &url.URL{Scheme: "http", Host: upstream}
	rp := &httputil.ReverseProxy{
		Transport:     s.transport,
		FlushInterval: -1,
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			req.Header.Set("X-Forwarded-Host", r.Host)
			req.Header.Set("X-Forwarded-Proto", schemeOf(r))
		},
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, perr error) {
			if isClientGone(perr) {
				return
			}
			log.Printf("[proxy] service upstream error host=%s upstream=%s err=%v", r.Host, upstream, perr)
			writeUnhealthy(rw, r.Host)
		},
	}
	rp.ServeHTTP(w, r)
}

func (s *Server) lookup(r *http.Request) (*projects.Project, bool) {
	proj, err := s.router.registry.Resolve(r.Context(), r.Host)
	if err != nil {
		return nil, false
	}
	return proj, true
}

func isPageNavigation(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}
	if isWebSocket(r) {
		return false
	}
	if dest := r.Header.Get("Sec-Fetch-Dest"); dest != "" {
		return dest == "document" || dest == "iframe"
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "text/html")
}

func (s *Server) renderWake(w http.ResponseWriter, r *http.Request, proj *projects.Project) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprint(w, wakePage(proj))
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
	if errors.Is(err, ErrUnhealthy) || errors.Is(err, ErrNoPort) {
		log.Printf("[proxy] route degraded host=%s err=%v", r.Host, err)
		s.renderRecovery(w, r)
		return
	}
	log.Printf("[proxy] route error host=%s err=%v", r.Host, err)
	s.renderRecovery(w, r)
}

func (s *Server) renderRecovery(w http.ResponseWriter, r *http.Request) {
	if s.recovery == nil {
		writeUnhealthy(w, r.Host)
		return
	}
	proj, err := s.router.registry.Resolve(r.Context(), r.Host)
	if err != nil {
		writeNotRegistered(w, r.Host)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusBadGateway)
	fmt.Fprint(w, recoveryPage(proj))
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
