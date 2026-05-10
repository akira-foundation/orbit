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
