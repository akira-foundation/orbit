package postgres

import (
	"context"
	"testing"
	"time"
)

func TestEnsureDatabaseUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := EnsureDatabase(ctx, "127.0.0.1", 1, "app"); err == nil {
		t.Fatal("expected connection error")
	}
}
