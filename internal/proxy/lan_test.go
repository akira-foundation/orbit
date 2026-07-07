package proxy

import (
	"net/http/httptest"
	"testing"
)

func TestRewriteShareHostFromQuery(t *testing.T) {
	r := httptest.NewRequest("GET", "http://192.168.1.42:2080/?__orbit=shop", nil)
	rewriteShareHost(r, "orbit.test")
	if r.Host != "shop.orbit.test" {
		t.Fatalf("Host = %q, want shop.orbit.test", r.Host)
	}
}

func TestRewriteShareHostNoTokenLeavesHost(t *testing.T) {
	r := httptest.NewRequest("GET", "http://192.168.1.42:2080/foo", nil)
	rewriteShareHost(r, "orbit.test")
	if r.Host != "192.168.1.42:2080" {
		t.Fatalf("Host = %q, want unchanged", r.Host)
	}
}
