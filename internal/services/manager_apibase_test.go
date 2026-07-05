package services

import (
	"testing"
)

func TestAPIBaseOnlyWhenRunning(t *testing.T) {
	store := newTestDB(t)
	acq := NewAcquirer(store, t.TempDir())
	resolver := NewResolver(store, AllowAll())
	m := NewManager(acq, resolver, LoadConfig(t.TempDir()), t.TempDir())

	if _, ok := m.APIBase("mailpit"); ok {
		t.Fatalf("expected no base when not managed-running")
	}
	if _, ok := m.APIBase("does-not-exist"); ok {
		t.Fatalf("expected no base for unknown engine")
	}
}
