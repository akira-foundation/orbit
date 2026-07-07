package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInjectAfterHead(t *testing.T) {
	out := string(injectAfterHead([]byte("<html><head><title>x</title></head><body></body></html>")))
	if !strings.Contains(out, "/__port/") {
		t.Fatalf("script not injected: %s", out)
	}
	if strings.Index(out, "<head>")+len("<head>") != strings.Index(out, "<script>") {
		t.Fatalf("script not placed right after <head>: %s", out)
	}
}

func TestInjectNoHeadPrepends(t *testing.T) {
	out := string(injectAfterHead([]byte("<div>hi</div>")))
	if !strings.HasPrefix(out, "<script>") {
		t.Fatalf("expected script prepended, got %s", out)
	}
}

func TestIsShareRequest(t *testing.T) {
	q := httptest.NewRequest("GET", "http://h/?__orbit=shop", nil)
	if !isShareRequest(q) {
		t.Fatal("query request should be share")
	}
	c := httptest.NewRequest("GET", "http://h/assets/x.js", nil)
	c.AddCookie(&http.Cookie{Name: "orbit_share", Value: "shop"})
	if !isShareRequest(c) {
		t.Fatal("cookie request should be share")
	}
	plain := httptest.NewRequest("GET", "http://h/", nil)
	if isShareRequest(plain) {
		t.Fatal("plain request should not be share")
	}
}
