package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInjectAfterHead(t *testing.T) {
	out := string(injectAfterHead([]byte("<html><head><title>x</title></head><body></body></html>"), "<script>S</script>"))
	if strings.Index(out, "<head>")+len("<head>") != strings.Index(out, "<script>S</script>") {
		t.Fatalf("script not placed right after <head>: %s", out)
	}
}

func TestInjectNoHeadPrepends(t *testing.T) {
	out := string(injectAfterHead([]byte("<div>hi</div>"), "<script>S</script>"))
	if !strings.HasPrefix(out, "<script>S</script>") {
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
	if isShareRequest(httptest.NewRequest("GET", "http://h/", nil)) {
		t.Fatal("plain request should not be share")
	}
}

func TestSplitPrefixPath(t *testing.T) {
	head, tail := splitPrefixPath("/__port/3000/auth/refresh", "/__port/")
	if head != "3000" || tail != "/auth/refresh" {
		t.Fatalf("got %q %q", head, tail)
	}
	head, tail = splitPrefixPath("/__proj/bu-payment-api", "/__proj/")
	if head != "bu-payment-api" || tail != "/" {
		t.Fatalf("got %q %q", head, tail)
	}
}

func TestRewriteProjPath(t *testing.T) {
	r := httptest.NewRequest("GET", "http://h/__proj/nosferry.com/api/x", nil)
	rewriteProjPath(r, "orbit.test")
	if r.Host != "nosferry.com.orbit.test" || r.URL.Path != "/api/x" {
		t.Fatalf("host=%q path=%q", r.Host, r.URL.Path)
	}
}
