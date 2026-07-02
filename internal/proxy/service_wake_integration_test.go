package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeHTTPRendersServiceWakeWhenDown(t *testing.T) {
	tbl := NewServiceTable("orbit.test", map[string]ServiceRoute{
		"mail": {Engine: "mailpit", DisplayName: "Mailpit", Upstream: "127.0.0.1:1"},
	})
	srv := NewServerWithOptions(Options{
		Addr:           "127.0.0.1:0",
		Services:       tbl,
		ServiceStarter: &stubStarter{},
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "mail.orbit.test"
	req.Header.Set("Accept", "text/html")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Start Mailpit") {
		t.Fatalf("wake page missing start button: %s", body[:min(200, len(body))])
	}
	if !strings.Contains(body, servicePrefix) || !strings.Contains(body, "/ping") {
		t.Fatal("wake page missing ping poll")
	}
}

func TestServeHTTPDispatchesServiceStart(t *testing.T) {
	tbl := NewServiceTable("orbit.test", map[string]ServiceRoute{
		"mail": {Engine: "mailpit", DisplayName: "Mailpit", Upstream: "127.0.0.1:1"},
	})
	starter := &stubStarter{}
	srv := NewServerWithOptions(Options{Addr: "127.0.0.1:0", Services: tbl, ServiceStarter: starter})

	req := httptest.NewRequest(http.MethodPost, servicePrefix+"/start", nil)
	req.Host = "mail.orbit.test"
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted || len(starter.started) != 1 {
		t.Fatalf("expected start dispatch, code=%d started=%v", rec.Code, starter.started)
	}
}
