package proxy

import (
	"crypto/tls"
	"net/http"
	"time"
)

func rewriteShareHost(r *http.Request, suffix string) {
	slug := r.URL.Query().Get("__orbit")
	if slug == "" || suffix == "" {
		return
	}
	r.Host = slug + "." + suffix
}

func (s *Server) EnableLAN(addr, certFile, keyFile string) error {
	if s.lan != nil {
		return nil
	}
	s.lan = &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
	}
	cache := newCertCache(certFile, keyFile)
	s.lan.TLSConfig = &tls.Config{GetCertificate: cache.GetCertificate}
	go func(srv *http.Server) {
		_ = srv.ListenAndServeTLS("", "")
	}(s.lan)
	return nil
}

func (s *Server) DisableLAN() error {
	if s.lan == nil {
		return nil
	}
	err := s.lan.Close()
	s.lan = nil
	return err
}
