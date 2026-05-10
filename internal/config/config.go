package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	DataDir      string
	DBPath       string
	DomainSuffix string
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
	}, nil
}

func (c *Config) DomainFor(slug string) string {
	return slug + "." + c.DomainSuffix
}
