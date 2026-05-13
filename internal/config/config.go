package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	DataDir              string `json:"-"`
	DBPath               string `json:"-"`
	DomainSuffix    string `json:"domainSuffix"`
	ProxyAddr       string `json:"proxyAddr"`
	ProxyTLSAddr    string `json:"proxyTlsAddr"`
	PublicPort      string `json:"publicPort"`
	PublicTLSPort   string `json:"publicTlsPort"`
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
	c := &Config{
		DataDir:      dir,
		DBPath:       filepath.Join(dir, "orbit.db"),
		DomainSuffix:  "orbit.test",
		ProxyAddr:     "127.0.0.1:2080",
		ProxyTLSAddr:  "127.0.0.1:2443",
		PublicPort:    "80",
		PublicTLSPort: "443",
	}

	configPath := filepath.Join(dir, "orbit.json")
	if data, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(data, c)
	}

	return c, nil
}

func (c *Config) Save() error {
	configPath := filepath.Join(c.DataDir, "orbit.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0o644)
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
