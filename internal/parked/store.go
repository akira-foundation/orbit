package parked

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type State struct {
	Folders    []string          `json:"folders"`
	Registered map[string]string `json:"registered"`
}

type Store struct {
	mu   sync.Mutex
	dir  string
	data State
}

func LoadStore(dir string) *Store {
	s := &Store{dir: dir, data: State{Registered: map[string]string{}}}
	if raw, err := os.ReadFile(filepath.Join(dir, "parked.json")); err == nil {
		_ = json.Unmarshal(raw, &s.data)
	}
	if s.data.Registered == nil {
		s.data.Registered = map[string]string{}
	}
	return s
}

func (s *Store) persist() error {
	data, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "parked.json"), data, 0o644)
}

func (s *Store) Folders() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string{}, s.data.Folders...)
}

func (s *Store) AddFolder(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, f := range s.data.Folders {
		if f == path {
			return nil
		}
	}
	s.data.Folders = append(s.data.Folders, path)
	return s.persist()
}

func (s *Store) RemoveFolder(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.data.Folders[:0]
	for _, f := range s.data.Folders {
		if f != path {
			next = append(next, f)
		}
	}
	s.data.Folders = next
	return s.persist()
}

func (s *Store) MarkRegistered(path, projectID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.Registered[path] = projectID
	return s.persist()
}

func (s *Store) Unregister(path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data.Registered, path)
	return s.persist()
}

func (s *Store) Registered() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]string, len(s.data.Registered))
	for k, v := range s.data.Registered {
		out[k] = v
	}
	return out
}

func (s *Store) AutoAddedIDs() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, 0, len(s.data.Registered))
	for _, id := range s.data.Registered {
		out = append(out, id)
	}
	return out
}
