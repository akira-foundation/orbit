package proxy

import (
	"crypto/tls"
	"os"
	"sync"
)

type certCache struct {
	certPath string
	keyPath  string

	mu      sync.Mutex
	modTime int64
	cert    *tls.Certificate
}

func newCertCache(certPath, keyPath string) *certCache {
	return &certCache{certPath: certPath, keyPath: keyPath}
}

func (c *certCache) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fi, statErr := os.Stat(c.certPath)
	if statErr == nil && c.cert != nil && fi.ModTime().UnixNano() == c.modTime {
		return c.cert, nil
	}

	pair, err := tls.LoadX509KeyPair(c.certPath, c.keyPath)
	if err != nil {
		if c.cert != nil {
			return c.cert, nil
		}
		return nil, err
	}
	c.cert = &pair
	if statErr == nil {
		c.modTime = fi.ModTime().UnixNano()
	}
	return c.cert, nil
}
