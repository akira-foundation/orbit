package proxy

import "testing"

func TestRequestLogRecentNewestFirstAndCapped(t *testing.T) {
	l := NewRequestLog()
	for i := 0; i < maxRequestsPerProject+50; i++ {
		l.Record("p1", RequestEntry{Ts: int64(i), Method: "GET", Path: "/x", Status: 200})
	}
	l.Record("p2", RequestEntry{Ts: 999, Path: "/other"})

	got := l.Recent("p1")
	if len(got) != maxRequestsPerProject {
		t.Fatalf("len = %d, want %d", len(got), maxRequestsPerProject)
	}
	if got[0].Ts <= got[1].Ts {
		t.Fatalf("expected newest first: %d then %d", got[0].Ts, got[1].Ts)
	}
	if got[len(got)-1].Ts != int64(50) {
		t.Fatalf("oldest kept = %d, want 50 (older dropped)", got[len(got)-1].Ts)
	}
	if len(l.Recent("p2")) != 1 {
		t.Fatal("p2 isolated from p1")
	}
	if len(l.Recent("missing")) != 0 {
		t.Fatal("unknown project should be empty")
	}
}
