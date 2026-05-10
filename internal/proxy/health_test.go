package proxy

import (
	"context"
	"errors"
	"net"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestWaitTCP_HitsListener(t *testing.T) {
	srv := httptest.NewServer(nil)
	defer srv.Close()

	host, portStr, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	if err != nil {
		t.Fatalf("split: %v", err)
	}
	port, _ := strconv.Atoi(portStr)

	if err := WaitTCP(context.Background(), host, port, HealthOptions{
		Timeout:      time.Second,
		PollInterval: 50 * time.Millisecond,
		DialTimeout:  200 * time.Millisecond,
	}); err != nil {
		t.Fatalf("WaitTCP: %v", err)
	}
}

func TestWaitTCP_TimesOut(t *testing.T) {
	err := WaitTCP(context.Background(), "127.0.0.1", 1, HealthOptions{
		Timeout:      300 * time.Millisecond,
		PollInterval: 50 * time.Millisecond,
		DialTimeout:  100 * time.Millisecond,
	})
	if !errors.Is(err, ErrUnhealthy) {
		t.Fatalf("expected ErrUnhealthy, got %v", err)
	}
}
