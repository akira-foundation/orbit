package share

import "sync"

type State struct {
	mu      sync.Mutex
	enabled map[string]bool
}

func NewState() *State {
	return &State{enabled: map[string]bool{}}
}

func (s *State) Enable(projectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled[projectID] = true
}

func (s *State) Disable(projectID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.enabled, projectID)
}

func (s *State) Enabled(projectID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled[projectID]
}

func (s *State) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.enabled)
}
