package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type RuntimesConfig struct {
	PreferSystemNode bool `json:"preferSystemNode"`
	PreferSystemPHP  bool `json:"preferSystemPhp"`
}

type RuntimesConfigStore struct {
	mu  sync.Mutex
	dir string
	cfg RuntimesConfig
}

func LoadRuntimesConfig(dir string) *RuntimesConfigStore {
	s := &RuntimesConfigStore{dir: dir}
	if data, err := os.ReadFile(filepath.Join(dir, "runtimes.json")); err == nil {
		_ = json.Unmarshal(data, &s.cfg)
	}
	return s
}

func (s *RuntimesConfigStore) Get() RuntimesConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *RuntimesConfigStore) Save(next RuntimesConfig) error {
	s.mu.Lock()
	s.cfg = next
	dir := s.dir
	data, err := json.MarshalIndent(next, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "runtimes.json"), data, 0o644)
}

func (s *RuntimesConfigStore) PreferSystemNode() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.PreferSystemNode
}

func (s *RuntimesConfigStore) PreferSystemPHP() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.PreferSystemPHP
}
