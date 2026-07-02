package services

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	AutoManage      bool            `json:"autoManage"`
	IdleStopMinutes int             `json:"idleStopMinutes"`
	Defaults        map[string]bool `json:"defaults"`
}

type ConfigStore struct {
	mu  sync.Mutex
	dir string
	cfg Config
}

func LoadConfig(dir string) *ConfigStore {
	s := &ConfigStore{
		dir: dir,
		cfg: Config{AutoManage: true, Defaults: map[string]bool{}},
	}
	if data, err := os.ReadFile(filepath.Join(dir, "services.json")); err == nil {
		_ = json.Unmarshal(data, &s.cfg)
	}
	if s.cfg.Defaults == nil {
		s.cfg.Defaults = map[string]bool{}
	}
	return s
}

func (s *ConfigStore) Get() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.copy()
}

func (s *ConfigStore) copy() Config {
	defaults := make(map[string]bool, len(s.cfg.Defaults))
	for k, v := range s.cfg.Defaults {
		defaults[k] = v
	}
	return Config{
		AutoManage:      s.cfg.AutoManage,
		IdleStopMinutes: s.cfg.IdleStopMinutes,
		Defaults:        defaults,
	}
}

func (s *ConfigStore) Save(next Config) error {
	s.mu.Lock()
	if next.Defaults == nil {
		next.Defaults = map[string]bool{}
	}
	s.cfg = next
	dir := s.dir
	data, err := json.MarshalIndent(s.copy(), "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "services.json"), data, 0o644)
}

func (s *ConfigStore) AutoManage() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.AutoManage
}

func (s *ConfigStore) IdleStopMinutes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.IdleStopMinutes
}

func (s *ConfigStore) DefaultEngines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []string
	for k, v := range s.cfg.Defaults {
		if v {
			out = append(out, k)
		}
	}
	return out
}
