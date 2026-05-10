package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DataDir      string
	DBPath       string
	DomainSuffix string
	ProxyAddr    string
	PublicPort   string
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".orbit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Config{
		DataDir:      dir,
		DBPath:       filepath.Join(dir, "orbit.db"),
		DomainSuffix: "orbit.test",
		ProxyAddr:    "127.0.0.1:2080",
		PublicPort:   "80",
	}, nil
}

func (c *Config) DomainFor(slug string) string {
	return slug + "." + c.DomainSuffix
}

func (c *Config) ProxyPort() string {
	if c.PublicPort != "" {
		return c.PublicPort
	}
	if c.ProxyAddr == "" {
		return "80"
	}
	addr := c.ProxyAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[i+1:]
		}
	}
	return "80"
}

func (c *Config) URLFor(domain string) string {
	port := c.ProxyPort()
	if port == "80" {
		return "http://" + domain
	}
	return "http://" + domain + ":" + port
}
