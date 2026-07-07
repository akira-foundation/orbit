package tls

import (
	"context"
	"path/filepath"
	"time"
)

func LeafExpiry(dir string) (time.Time, bool) {
	cert, err := readCert(filepath.Join(dir, "orbit-leaf.pem"))
	if err != nil {
		return time.Time{}, false
	}
	return cert.NotAfter, true
}

func StartSweep(ctx context.Context, dir string, emit func(name string, data ...any)) {
	check := func() {
		exp, ok := LeafExpiry(dir)
		if !ok {
			return
		}
		if time.Until(exp) < 30*24*time.Hour {
			emit("tls:expiring", map[string]any{"expiresAt": exp.Format(time.RFC3339)})
		}
	}
	check()
	tick := time.NewTicker(6 * time.Hour)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			check()
		}
	}
}
