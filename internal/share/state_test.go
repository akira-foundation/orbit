package share

import "testing"

func TestStateToggle(t *testing.T) {
	s := NewState()
	if s.Enabled("p1") || s.Count() != 0 {
		t.Fatal("should start empty")
	}
	s.Enable("p1")
	if !s.Enabled("p1") || s.Count() != 1 {
		t.Fatalf("enable failed: %v %d", s.Enabled("p1"), s.Count())
	}
	s.Enable("p1")
	if s.Count() != 1 {
		t.Fatalf("re-enable should be idempotent, count=%d", s.Count())
	}
	s.Disable("p1")
	if s.Enabled("p1") || s.Count() != 0 {
		t.Fatal("disable failed")
	}
}
