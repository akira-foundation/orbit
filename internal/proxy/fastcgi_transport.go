package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type fcgiTransport struct {
	SockPath string
	DocRoot  string
}

// exported so other packages' integration tests can construct the unexported type
func NewFastCGITransport(sockPath, docRoot string) http.RoundTripper {
	return &fcgiTransport{SockPath: sockPath, DocRoot: docRoot}
}

func (t *fcgiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	scriptFile, static := resolveScriptTarget(t.DocRoot, req.URL.Path)
	if static {
		return serveStaticFile(scriptFile)
	}

	body := req.Body
	if body == nil {
		body = http.NoBody
	}
	pairs := t.buildParams(req, scriptFile)
	resp, err := doFastCGI(t.SockPath, pairs, body)
	if err != nil {
		return nil, err
	}
	if len(resp.stderr) > 0 {
		log.Printf("[proxy] php-fpm stderr sock=%s: %s", t.SockPath, resp.stderr)
	}
	return parseCGIResponse(resp.stdout, req)
}

func (t *fcgiTransport) buildParams(req *http.Request, scriptFile string) [][2]string {
	contentLength := req.Header.Get("Content-Length")
	if contentLength == "" && req.ContentLength >= 0 {
		contentLength = strconv.FormatInt(req.ContentLength, 10)
	}
	pairs := [][2]string{
		{"GATEWAY_INTERFACE", "CGI/1.1"},
		{"SERVER_SOFTWARE", "orbit"},
		{"REQUEST_METHOD", req.Method},
		{"SCRIPT_FILENAME", scriptFile},
		{"SCRIPT_NAME", "/" + filepath.Base(scriptFile)},
		{"QUERY_STRING", req.URL.RawQuery},
		{"REQUEST_URI", req.URL.RequestURI()},
		{"DOCUMENT_ROOT", t.DocRoot},
		{"DOCUMENT_URI", req.URL.Path},
		{"SERVER_PROTOCOL", req.Proto},
		{"SERVER_NAME", req.Host},
		{"CONTENT_TYPE", req.Header.Get("Content-Type")},
		{"CONTENT_LENGTH", contentLength},
		{"HTTPS", httpsFlag(req)},
	}
	if host, port, ok := splitRemoteAddr(req.RemoteAddr); ok {
		pairs = append(pairs, [2]string{"REMOTE_ADDR", host}, [2]string{"REMOTE_PORT", port})
	}
	for name, values := range req.Header {
		key := "HTTP_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
		pairs = append(pairs, [2]string{key, strings.Join(values, ", ")})
	}
	return pairs
}

func httpsFlag(req *http.Request) string {
	if req.TLS != nil {
		return "on"
	}
	return ""
}

func splitRemoteAddr(addr string) (host, port string, ok bool) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return "", "", false
	}
	return addr[:i], addr[i+1:], true
}

// mirrors nginx's try_files $uri /index.php
func resolveScriptTarget(docRoot, urlPath string) (scriptFile string, static bool) {
	clean := filepath.Clean("/" + urlPath)
	candidate := filepath.Join(docRoot, filepath.FromSlash(clean))
	rootClean := filepath.Clean(docRoot)
	if candidate != rootClean && !strings.HasPrefix(candidate, rootClean+string(filepath.Separator)) {
		candidate = rootClean
	}
	if strings.EqualFold(filepath.Ext(candidate), ".php") {
		return filepath.Join(docRoot, "index.php"), false
	}
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
		return candidate, true
	}
	return filepath.Join(docRoot, "index.php"), false
}

func serveStaticFile(path string) (*http.Response, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("fastcgi: open static file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	header := http.Header{}
	if ct := mimeTypeByExtension(path); ct != "" {
		header.Set("Content-Type", ct)
	}
	return &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        header,
		Body:          f,
		ContentLength: info.Size(),
	}, nil
}

func parseCGIResponse(raw []byte, req *http.Request) (*http.Response, error) {
	tp := textproto.NewReader(bufio.NewReader(bytes.NewReader(raw)))
	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("fastcgi: parse response headers: %w", err)
	}
	header := http.Header(mimeHeader)

	status := http.StatusOK
	statusText := "OK"
	if line := header.Get("Status"); line != "" {
		header.Del("Status")
		parts := strings.SplitN(line, " ", 2)
		if code, cerr := strconv.Atoi(parts[0]); cerr == nil {
			status = code
			if len(parts) > 1 {
				statusText = parts[1]
			}
		}
	}

	bodyStart := headerBlockLen(raw)
	body := raw[bodyStart:]

	return &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, statusText),
		Proto:         req.Proto,
		ProtoMajor:    req.ProtoMajor,
		ProtoMinor:    req.ProtoMinor,
		Header:        header,
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}, nil
}

func headerBlockLen(raw []byte) int {
	if i := bytes.Index(raw, []byte("\r\n\r\n")); i >= 0 {
		return i + 4
	}
	if i := bytes.Index(raw, []byte("\n\n")); i >= 0 {
		return i + 2
	}
	return len(raw)
}

func mimeTypeByExtension(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".js", ".mjs":
		return "application/javascript; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".woff2":
		return "font/woff2"
	case ".ico":
		return "image/x-icon"
	default:
		return ""
	}
}
