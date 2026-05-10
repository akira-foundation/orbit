package proxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"
)

var ErrUnhealthy = errors.New("proxy: target failed healthcheck")

type HealthOptions struct {
	Timeout      time.Duration
	PollInterval time.Duration
	DialTimeout  time.Duration
}

func defaultHealthOptions() HealthOptions {
	return HealthOptions{
		Timeout:      45 * time.Second,
		PollInterval: 250 * time.Millisecond,
		DialTimeout:  500 * time.Millisecond,
	}
}

func WaitTCP(ctx context.Context, host string, port int, opts HealthOptions) error {
	if opts.Timeout == 0 {
		opts = defaultHealthOptions()
	}
	if opts.PollInterval == 0 {
		opts.PollInterval = 250 * time.Millisecond
	}
	if opts.DialTimeout == 0 {
		opts.DialTimeout = 500 * time.Millisecond
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	deadline := time.Now().Add(opts.Timeout)
	ctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	for {
		if dialOnce(ctx, addr, opts.DialTimeout) == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %s after %s", ErrUnhealthy, addr, opts.Timeout)
		case <-time.After(opts.PollInterval):
		}
	}
}

// dialableHost returns the first host (127.0.0.1 / ::1 / localhost) that
// accepts a TCP connection on port within HealthOptions.Timeout. Frameworks
// vary: Astro binds to "localhost" which on macOS may resolve to ::1 only;
// most others bind 127.0.0.1. Try all in parallel-ish polling.
func dialableHost(ctx context.Context, port int, opts HealthOptions) (string, error) {
	if opts.Timeout == 0 {
		opts = defaultHealthOptions()
	}
	candidates := []string{"127.0.0.1", "::1", "localhost"}
	deadline := time.Now().Add(opts.Timeout)
	dctx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	for {
		for _, h := range candidates {
			addr := net.JoinHostPort(h, strconv.Itoa(port))
			if dialOnce(dctx, addr, opts.DialTimeout) == nil {
				return h, nil
			}
		}
		select {
		case <-dctx.Done():
			return "", fmt.Errorf("%w: port %d after %s", ErrUnhealthy, port, opts.Timeout)
		case <-time.After(opts.PollInterval):
		}
	}
}

func dialOnce(ctx context.Context, addr string, timeout time.Duration) error {
	dctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	d := net.Dialer{}
	conn, err := d.DialContext(dctx, "tcp", addr)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
