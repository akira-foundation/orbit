package proxy

import "sync"

const maxRequestsPerProject = 200

type RequestEntry struct {
	Ts         int64   `json:"ts"`
	Method     string  `json:"method"`
	Path       string  `json:"path"`
	Status     int     `json:"status"`
	DurationMs float64 `json:"durationMs"`
	Bytes      int64   `json:"bytes"`
	WS         bool    `json:"ws"`
}

type RequestLog struct {
	mu sync.Mutex
	by map[string][]RequestEntry
}

func NewRequestLog() *RequestLog {
	return &RequestLog{by: map[string][]RequestEntry{}}
}

func (l *RequestLog) Record(projectID string, e RequestEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	buf := l.by[projectID]
	buf = append(buf, e)
	if len(buf) > maxRequestsPerProject {
		buf = buf[len(buf)-maxRequestsPerProject:]
	}
	l.by[projectID] = buf
}

func (l *RequestLog) Recent(projectID string) []RequestEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	buf := l.by[projectID]
	out := make([]RequestEntry, len(buf))
	for i, e := range buf {
		out[len(buf)-1-i] = e
	}
	return out
}
