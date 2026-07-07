package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
)

const shareScript = `<script>(function(){var rw=function(u){try{if(typeof u!=='string')return u;return u.replace(/https?:\/\/(?:localhost|127\.0\.0\.1):(\d+)/g,'/__port/$1')}catch(e){return u}};var of=window.fetch;if(of){window.fetch=function(i,init){if(i&&typeof i==='object'&&i.url){return of(new Request(rw(i.url),i),init)}return of(rw(i),init)}}var xo=XMLHttpRequest.prototype.open;XMLHttpRequest.prototype.open=function(){if(arguments.length>1){arguments[1]=rw(arguments[1])}return xo.apply(this,arguments)}})();</script>`

func isShareRequest(r *http.Request) bool {
	if r.URL.Query().Get("__orbit") != "" {
		return true
	}
	_, err := r.Cookie(shareCookie)
	return err == nil
}

func injectShareResponse(resp *http.Response) error {
	if resp.Header.Get("Content-Encoding") != "" {
		return nil
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return nil
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return err
	}
	out := injectAfterHead(body)
	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
	resp.Header.Set("Content-Length", strconv.Itoa(len(out)))
	return nil
}

func injectAfterHead(body []byte) []byte {
	lower := bytes.ToLower(body)
	idx := bytes.Index(lower, []byte("<head>"))
	if idx < 0 {
		return append([]byte(shareScript), body...)
	}
	at := idx + len("<head>")
	out := make([]byte, 0, len(body)+len(shareScript))
	out = append(out, body[:at]...)
	out = append(out, []byte(shareScript)...)
	out = append(out, body[at:]...)
	return out
}

func (s *Server) proxyToLocalPort(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/__port/")
	slash := strings.IndexByte(rest, '/')
	port := rest
	tail := "/"
	if slash >= 0 {
		port = rest[:slash]
		tail = rest[slash:]
	}
	if _, err := strconv.Atoi(port); err != nil {
		http.Error(w, "bad port", http.StatusBadRequest)
		return
	}
	target := &url.URL{Scheme: "http", Host: "127.0.0.1:" + port}
	rp := &httputil.ReverseProxy{
		Transport:     s.transport,
		FlushInterval: -1,
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = tail
			req.Host = "localhost:" + port
		},
	}
	rp.ServeHTTP(w, r)
}
