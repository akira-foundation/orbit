package services

import (
	"testing"
)

func TestAPIBaseOnlyWhenRunning(t *testing.T) {
	m := NewManager(nil, nil, nil, t.TempDir())
	if _, ok := m.APIBase("mailpit"); ok {
		t.Fatalf("expected no base when stopped")
	}
	if _, ok := m.APIBase("does-not-exist"); ok {
		t.Fatalf("expected no base for unknown engine")
	}
}
