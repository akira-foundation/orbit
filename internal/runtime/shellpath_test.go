package runtime

import (
	"strings"
	"testing"
)

func TestMergePathListDedupsAndOrders(t *testing.T) {
	got := mergePathList("/usr/bin:/bin", "/opt/homebrew/bin:/usr/bin")
	if got != "/usr/bin:/bin:/opt/homebrew/bin" {
		t.Fatalf("merge = %q", got)
	}
}

func TestWithLoginPathAddsWhenMissing(t *testing.T) {
	env := withLoginPath([]string{"FOO=bar"})
	var pathVal string
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			pathVal = kv
		}
	}
	if pathVal == "" {
		t.Fatal("expected PATH to be added")
	}
}

func TestWithLoginPathMergesExisting(t *testing.T) {
	env := withLoginPath([]string{"PATH=/custom/bin"})
	for _, kv := range env {
		if strings.HasPrefix(kv, "PATH=") {
			if !strings.Contains(kv, "/custom/bin") {
				t.Fatalf("lost existing PATH entry: %q", kv)
			}
			return
		}
	}
	t.Fatal("no PATH in env")
}
