package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRewriteShareHostFromQuery(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://192.168.1.42:2080/?__orbit=shop", nil)
	rewriteShareHost(w, r, "orbit.test")
	if r.Host != "shop.orbit.test" {
		t.Fatalf("Host = %q, want shop.orbit.test", r.Host)
	}
	if !hasShareCookie(w, "shop") {
		t.Fatal("expected orbit_share cookie to be set")
	}
}

func TestRewriteShareHostFromCookie(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://192.168.1.42:2080/assets/app.js", nil)
	r.AddCookie(&http.Cookie{Name: "orbit_share", Value: "shop"})
	rewriteShareHost(w, r, "orbit.test")
	if r.Host != "shop.orbit.test" {
		t.Fatalf("Host = %q, want shop.orbit.test (from cookie)", r.Host)
	}
}

func TestRewriteShareHostNoTokenLeavesHost(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "http://192.168.1.42:2080/foo", nil)
	rewriteShareHost(w, r, "orbit.test")
	if r.Host != "192.168.1.42:2080" {
		t.Fatalf("Host = %q, want unchanged", r.Host)
	}
}

func hasShareCookie(w *httptest.ResponseRecorder, val string) bool {
	for _, c := range w.Result().Cookies() {
		if c.Name == "orbit_share" && c.Value == val {
			return true
		}
	}
	return false
}
