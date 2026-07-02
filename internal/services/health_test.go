package services

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestWaitDialableSucceeds(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	if err := waitDialable(context.Background(), ln.Addr().String(), 2*time.Second); err != nil {
		t.Fatalf("expected dialable, got %v", err)
	}
}

func TestWaitDialableTimesOut(t *testing.T) {
	err := waitDialable(context.Background(), "127.0.0.1:1", 300*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
