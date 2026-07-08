package copilot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
}

type ConfigStore struct {
	mu  sync.Mutex
	dir string
	cfg Config
}

func LoadConfig(dir string) *ConfigStore {
	s := &ConfigStore{dir: dir, cfg: Config{Provider: "auto"}}
	if data, err := os.ReadFile(filepath.Join(dir, "copilot.json")); err == nil {
		_ = json.Unmarshal(data, &s.cfg)
	}
	if s.cfg.Provider == "" {
		s.cfg.Provider = "auto"
	}
	return s
}

func (s *ConfigStore) Get() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *ConfigStore) Save(next Config) error {
	s.mu.Lock()
	s.cfg = next
	dir := s.dir
	data, err := json.MarshalIndent(next, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "copilot.json"), data, 0o644)
}
