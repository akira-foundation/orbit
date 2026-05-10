package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"orbit-app/internal/projects"
)

type LogLine struct {
	Timestamp string `json:"timestamp"`
	Stream    string `json:"stream"`
	Text      string `json:"text"`
}

type Snapshot struct {
	ProjectID    string          `json:"projectId"`
	Status       projects.Status `json:"status"`
	PID          int             `json:"pid"`
	Port         int             `json:"port"`
	StartedAt    string          `json:"startedAt"`
	ReadyAt      string          `json:"readyAt,omitempty"`
	UptimeMs     int64           `json:"uptimeMs"`
	StartupMs    int64           `json:"startupMs,omitempty"`
	Attempts     int             `json:"attempts,omitempty"`
	Conns        int             `json:"conns"`
	LastActivity string          `json:"lastActivity"`
	Error        string          `json:"error,omitempty"`
}

type Session struct {
	mu sync.RWMutex

	projectID    string
	sessionID    string
	status       projects.Status
	pid          int
	port         int
	startedAt    time.Time
	readyAt      time.Time
	lastActivity time.Time
	errMsg       string

	// Self-healing accounting. attempts counts consecutive failures since
	// the last successful run; reaches 0 again as soon as the session
	// becomes Running. wasRunning flips true once a session reached
	// Running so we can tell premature startup failures apart from real
	// runtime crashes.
	attempts   int
	wasRunning bool

	logs     []LogLine
	logCap   int
	logTotal uint64

	stopCh chan struct{}
	killFn func()
	pgid   int
}

func newSession(projectID string, logCap int) *Session {
	return &Session{
		projectID: projectID,
		sessionID: newSessionID(),
		status:    projects.StatusStarting,
		startedAt: time.Now().UTC(),
		logs:      make([]LogLine, 0, logCap),
		logCap:    logCap,
		stopCh:    make(chan struct{}),
	}
}

// newSessionID returns a short random hex token used as session_id in
// runtime_logs. 8 bytes of entropy is plenty to disambiguate concurrent runs.
func newSessionID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (s *Session) setStatus(st projects.Status) {
	s.mu.Lock()
	s.status = st
	s.lastActivity = time.Now().UTC()
	if st == projects.StatusRunning {
		if s.readyAt.IsZero() {
			s.readyAt = time.Now().UTC()
		}
		s.wasRunning = true
		s.attempts = 0
	}
	s.mu.Unlock()
}

func (s *Session) setPID(pid int) {
	s.mu.Lock()
	s.pid = pid
	s.mu.Unlock()
}

func (s *Session) setPort(p int) {
	s.mu.Lock()
	s.port = p
	s.lastActivity = time.Now().UTC()
	s.mu.Unlock()
}

func (s *Session) setError(err string) {
	s.mu.Lock()
	s.errMsg = err
	s.status = projects.StatusError
	s.lastActivity = time.Now().UTC()
	s.mu.Unlock()
}

func (s *Session) markStopped() {
	s.mu.Lock()
	s.status = projects.StatusStopped
	s.lastActivity = time.Now().UTC()
	s.mu.Unlock()
}

func (s *Session) appendLog(line LogLine) {
	s.mu.Lock()
	if len(s.logs) >= s.logCap {
		copy(s.logs, s.logs[1:])
		s.logs = s.logs[:len(s.logs)-1]
	}
	s.logs = append(s.logs, line)
	s.logTotal++
	s.lastActivity = time.Now().UTC()
	s.mu.Unlock()
}

func (s *Session) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	uptime := int64(0)
	if s.status == projects.StatusRunning || s.status == projects.StatusStarting {
		uptime = time.Since(s.startedAt).Milliseconds()
	}
	startupMs := int64(0)
	readyStr := ""
	if !s.readyAt.IsZero() {
		startupMs = s.readyAt.Sub(s.startedAt).Milliseconds()
		readyStr = s.readyAt.Format(time.RFC3339)
	}
	return Snapshot{
		ProjectID:    s.projectID,
		Status:       s.status,
		PID:          s.pid,
		Port:         s.port,
		StartedAt:    s.startedAt.Format(time.RFC3339),
		ReadyAt:      readyStr,
		UptimeMs:     uptime,
		StartupMs:    startupMs,
		Attempts:     s.attempts,
		LastActivity: s.lastActivity.Format(time.RFC3339),
		Error:        s.errMsg,
	}
}

func (s *Session) Logs() []LogLine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]LogLine, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *Session) Status() projects.Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Session) PID() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.pid
}

func (s *Session) Port() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.port
}

func (s *Session) LastActivity() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastActivity
}

func (s *Session) Attempts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.attempts
}

func (s *Session) WasRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.wasRunning
}

func (s *Session) bumpAttempts() int {
	s.mu.Lock()
	s.attempts++
	n := s.attempts
	s.mu.Unlock()
	return n
}
