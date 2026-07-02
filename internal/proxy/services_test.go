package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testTable() ServiceResolver {
	return NewServiceTable("orbit.test", map[string]ServiceRoute{
		"mail": {Engine: "mailpit", DisplayName: "Mailpit", Upstream: "127.0.0.2:8025"},
	})
}

func TestServiceTableResolvesReservedHost(t *testing.T) {
	tbl := testTable()
	route, ok := tbl.ResolveService("mail.orbit.test")
	if !ok || route.Upstream != "127.0.0.2:8025" || route.Engine != "mailpit" {
		t.Fatalf("resolve = %+v ok=%v", route, ok)
	}
	if _, ok := tbl.ResolveService("mail.orbit.test:443"); !ok {
		t.Fatal("expected host with port to resolve")
	}
	if _, ok := tbl.ResolveService("myapp.orbit.test"); ok {
		t.Fatal("non-reserved host should not resolve")
	}
}

type stubStarter struct {
	started []string
}

func (s *stubStarter) StartManual(_ context.Context, engine string) error {
	s.started = append(s.started, engine)
	return nil
}

func TestServiceHandlerStartTriggersEngine(t *testing.T) {
	starter := &stubStarter{}
	h := NewServiceHandler(testTable(), starter)

	req := httptest.NewRequest(http.MethodPost, servicePrefix+"/start", nil)
	req.Host = "mail.orbit.test"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d", rec.Code)
	}
	if len(starter.started) != 1 || starter.started[0] != "mailpit" {
		t.Fatalf("started = %v", starter.started)
	}
}

func TestServiceHandlerPingDownReturns503(t *testing.T) {
	down := NewServiceTable("orbit.test", map[string]ServiceRoute{
		"mail": {Engine: "mailpit", DisplayName: "Mailpit", Upstream: "127.0.0.1:1"},
	})
	h := NewServiceHandler(down, &stubStarter{})
	req := httptest.NewRequest(http.MethodGet, servicePrefix+"/ping", nil)
	req.Host = "mail.orbit.test"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for down upstream, got %d", rec.Code)
	}
}

func TestServiceHandlerPingUpReturns200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()
	upstream := srv.Listener.Addr().String()

	up := NewServiceTable("orbit.test", map[string]ServiceRoute{
		"mail": {Engine: "mailpit", DisplayName: "Mailpit", Upstream: upstream},
	})
	h := NewServiceHandler(up, &stubStarter{})
	req := httptest.NewRequest(http.MethodGet, servicePrefix+"/ping", nil)
	req.Host = "mail.orbit.test"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for reachable upstream, got %d", rec.Code)
	}
}
