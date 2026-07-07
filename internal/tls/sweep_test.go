package tls

import (
	"testing"
	"time"
)

func TestLeafExpiry(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	mat, err := Ensure("orbit.test", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	exp, ok := LeafExpiry(mat.Dir)
	if !ok {
		t.Fatal("expected a leaf expiry")
	}
	if time.Until(exp) < 300*24*time.Hour {
		t.Fatalf("expiry too soon: %v", exp)
	}
}

func TestLeafExpiryMissing(t *testing.T) {
	if _, ok := LeafExpiry(t.TempDir()); ok {
		t.Fatal("expected ok=false for missing leaf")
	}
}
