package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestEnabledForUnionOfTogglesAndYAML(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()
	_ = store.SetEnabled(ctx, "p1", "mailpit", true)

	dir := t.TempDir()
	r := NewResolver(store, AllowAll())

	engines, err := r.EnabledFor(ctx, "p1", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(engines) != 1 || engines[0] != "mailpit" {
		t.Fatalf("engines = %v", engines)
	}
}

func TestEnabledForYAMLAddsEngine(t *testing.T) {
	store := newTestDB(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".orbit.yaml"),
		[]byte("services:\n  mailpit: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(store, AllowAll())
	engines, err := r.EnabledFor(context.Background(), "p2", dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(engines) != 1 || engines[0] != "mailpit" {
		t.Fatalf("engines = %v", engines)
	}
}

func TestEnabledForDropsUnknownEngine(t *testing.T) {
	store := newTestDB(t)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, ".orbit.yaml"),
		[]byte("services:\n  ghost: true\n"), 0o644)
	r := NewResolver(store, AllowAll())
	engines, _ := r.EnabledFor(context.Background(), "p3", dir)
	if len(engines) != 0 {
		t.Fatalf("expected unknown engine dropped, got %v", engines)
	}
}

type denyAll struct{}

func (denyAll) Has(string) bool { return false }

func TestEnabledForEntitlementGate(t *testing.T) {
	store := newTestDB(t)
	ctx := context.Background()
	_ = store.SetEnabled(ctx, "p4", "mailpit", true)
	r := NewResolver(store, denyAll{})
	engines, _ := r.EnabledFor(ctx, "p4", t.TempDir())
	if len(engines) != 0 {
		t.Fatalf("expected gated out, got %v", engines)
	}
}
