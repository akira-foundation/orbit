package copilot

import (
	"strings"
	"testing"
)

func TestRecentErrorContextFindsLastError(t *testing.T) {
	lines := []string{}
	for i := 0; i < 50; i++ {
		lines = append(lines, "listening on :3000")
	}
	lines = append(lines, "TypeError: cannot read properties of undefined")
	lines = append(lines, "    at handler (src/routes/user.ts:42:11)")

	ctx, found := RecentErrorContext(lines)
	if !found {
		t.Fatal("expected an error match")
	}
	if !strings.Contains(ctx, "TypeError") || !strings.Contains(ctx, "user.ts:42") {
		t.Fatalf("context missing error lines: %q", ctx)
	}
}

func TestRecentErrorContextFallsBackToTail(t *testing.T) {
	lines := []string{}
	for i := 0; i < 100; i++ {
		lines = append(lines, "ok")
	}
	ctx, found := RecentErrorContext(lines)
	if found {
		t.Fatal("expected no error match")
	}
	if got := len(strings.Split(ctx, "\n")); got != fallbackTail {
		t.Fatalf("tail lines = %d", got)
	}
}

func TestResolveProvider(t *testing.T) {
	avail := []Provider{{ID: "claude"}, {ID: "codex"}}
	if id, err := ResolveProvider("auto", avail); err != nil || id != "claude" {
		t.Fatalf("auto = %q %v", id, err)
	}
	if id, err := ResolveProvider("codex", avail); err != nil || id != "codex" {
		t.Fatalf("codex = %q %v", id, err)
	}
	if _, err := ResolveProvider("gemini", avail); err == nil {
		t.Fatal("expected error for missing provider")
	}
	if _, err := ResolveProvider("auto", nil); err == nil {
		t.Fatal("expected error when nothing installed")
	}
}

func TestPromptsIncludeContext(t *testing.T) {
	p := ExplainPrompt("laravel", "boom")
	if !strings.Contains(p, "laravel") || !strings.Contains(p, "boom") {
		t.Fatalf("explain prompt = %q", p)
	}
	q := AskPrompt("", "logline", "why 500?")
	if !strings.Contains(q, "unknown") || !strings.Contains(q, "why 500?") {
		t.Fatalf("ask prompt = %q", q)
	}
	w := WhySlowPrompt("vite", SlowStats{ReqCount: 9, P95Ms: 420}, "logs")
	if !strings.Contains(w, "p95 420ms") || !strings.Contains(w, "vite") {
		t.Fatalf("why-slow prompt = %q", w)
	}
}

func TestConfigStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := LoadConfig(dir)
	if got := s.Get(); got.Enabled || got.Provider != "auto" {
		t.Fatalf("default = %+v", got)
	}
	if err := s.Save(Config{Enabled: true, Provider: "codex"}); err != nil {
		t.Fatal(err)
	}
	re := LoadConfig(dir)
	if got := re.Get(); !got.Enabled || got.Provider != "codex" {
		t.Fatalf("reloaded = %+v", got)
	}
}
