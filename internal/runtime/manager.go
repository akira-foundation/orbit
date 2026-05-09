package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"orbit-app/internal/projects"
)

type ProjectLookup interface {
	Get(ctx context.Context, id string) (*projects.Project, error)
	UpdateStatus(ctx context.Context, id string, status projects.Status) error
}

type Manager interface {
	Start(ctx context.Context, projectID string) error
	Stop(ctx context.Context, projectID string) error
	Restart(ctx context.Context, projectID string) error
	Status(projectID string) Snapshot
	Logs(projectID string) []LogLine
	GetStatus(ctx context.Context, projectID string) (projects.Status, error)
	StopAll()
	SetEmitter(e Emitter)
}

func New(projects ProjectLookup) Manager {
	return &manager{
		projects:  projects,
		emitter:   nopEmitter{},
		sessions:  make(map[string]*Session),
		stopGrace: 5 * time.Second,
		logCap:    2000,
	}
}

type manager struct {
	mu       sync.RWMutex
	projects ProjectLookup
	emitter  Emitter
	sessions map[string]*Session

	stopGrace time.Duration
	logCap    int
}

func (m *manager) SetEmitter(e Emitter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e == nil {
		m.emitter = nopEmitter{}
		return
	}
	m.emitter = e
}

func (m *manager) emit(name string, data ...any) {
	m.mu.RLock()
	e := m.emitter
	m.mu.RUnlock()
	e.Emit(name, data...)
}

func (m *manager) Start(ctx context.Context, projectID string) error {
	m.mu.Lock()
	if existing, ok := m.sessions[projectID]; ok {
		st := existing.Status()
		if st == projects.StatusStarting || st == projects.StatusRunning {
			m.mu.Unlock()
			return errors.New("runtime: already running")
		}
	}
	m.mu.Unlock()

	proj, err := m.projects.Get(ctx, projectID)
	if err != nil {
		return fmt.Errorf("runtime: load project: %w", err)
	}
	if proj.DevCommand == "" {
		return errors.New("runtime: project has no dev command")
	}

	env := append(os.Environ(),
		"FORCE_COLOR=1",
		"CI=false",
	)

	handle, err := spawnDevCommand(proj.Path, proj.DevCommand, env)
	if err != nil {
		return fmt.Errorf("runtime: spawn: %w", err)
	}

	sess := newSession(projectID, m.logCap)
	sess.setPID(handle.cmd.Process.Pid)

	m.mu.Lock()
	m.sessions[projectID] = sess
	m.mu.Unlock()

	_ = m.projects.UpdateStatus(ctx, projectID, projects.StatusStarting)
	m.emit(EvtStarting, StatusEvent{ProjectID: projectID, Snapshot: sess.Snapshot()})

	go m.readPipe(sess, handle.stdout, "stdout")
	go m.readPipe(sess, handle.stderr, "stderr")
	go m.supervise(sess, handle, proj)

	return nil
}

func (m *manager) Stop(ctx context.Context, projectID string) error {
	m.mu.RLock()
	sess, ok := m.sessions[projectID]
	m.mu.RUnlock()
	if !ok {
		_ = m.projects.UpdateStatus(ctx, projectID, projects.StatusStopped)
		return nil
	}

	if sess.killFn != nil {
		sess.killFn()
	}
	close(sess.stopCh)

	deadline := time.Now().Add(m.stopGrace + time.Second)
	for time.Now().Before(deadline) {
		if sess.Status() == projects.StatusStopped || sess.Status() == projects.StatusError {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	_ = m.projects.UpdateStatus(ctx, projectID, projects.StatusStopped)
	return nil
}

func (m *manager) Restart(ctx context.Context, projectID string) error {
	if err := m.Stop(ctx, projectID); err != nil {
		return err
	}
	time.Sleep(300 * time.Millisecond)
	return m.Start(ctx, projectID)
}

func (m *manager) Status(projectID string) Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[projectID]; ok {
		return sess.Snapshot()
	}
	return Snapshot{ProjectID: projectID, Status: projects.StatusStopped}
}

func (m *manager) Logs(projectID string) []LogLine {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[projectID]; ok {
		return sess.Logs()
	}
	return nil
}

func (m *manager) GetStatus(_ context.Context, projectID string) (projects.Status, error) {
	return m.Status(projectID).Status, nil
}

func (m *manager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	for id, s := range m.sessions {
		st := s.Status()
		if st == projects.StatusStarting || st == projects.StatusRunning {
			ids = append(ids, id)
		}
	}
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, id := range ids {
		_ = m.Stop(ctx, id)
	}
}

func (m *manager) readPipe(sess *Session, r io.Reader, stream string) {
	scanLines(r, func(text string) bool {
		line := LogLine{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Stream:    stream,
			Text:      text,
		}
		sess.appendLog(line)
		m.emit(EvtLog, LogEvent{ProjectID: sess.projectID, Line: line})

		if sess.Port() == 0 {
			if port := extractPort(text); port > 0 {
				sess.setPort(port)
			}
		}
		if sess.Status() == projects.StatusStarting && looksReady(text) {
			sess.setStatus(projects.StatusRunning)
			_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusRunning)
			m.emit(EvtRunning, StatusEvent{ProjectID: sess.projectID, Snapshot: sess.Snapshot()})
		}
		return true
	})
}

func (m *manager) supervise(sess *Session, handle *processHandle, _ *projects.Project) {
	sess.killFn = func() {
		_ = handle.kill()
	}

	doneCh := make(chan error, 1)
	go func() {
		doneCh <- handle.cmd.Wait()
	}()

	select {
	case err := <-doneCh:
		m.finalize(sess, err, false)
	case <-sess.stopCh:
		select {
		case err := <-doneCh:
			m.finalize(sess, err, true)
		case <-time.After(m.stopGrace):
			_ = handle.forceKill()
			err := <-doneCh
			m.finalize(sess, err, true)
		}
	}
}

func (m *manager) finalize(sess *Session, err error, requested bool) {
	if requested {
		sess.markStopped()
		m.emit(EvtStopped, StatusEvent{ProjectID: sess.projectID, Snapshot: sess.Snapshot()})
		_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusStopped)
		return
	}

	if err != nil {
		sess.setError(err.Error())
		sess.appendLog(LogLine{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Stream:    "system",
			Text:      "process exited: " + err.Error(),
		})
		m.emit(EvtError, StatusEvent{ProjectID: sess.projectID, Snapshot: sess.Snapshot()})
		_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusError)
		return
	}
	sess.markStopped()
	m.emit(EvtStopped, StatusEvent{ProjectID: sess.projectID, Snapshot: sess.Snapshot()})
	_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusStopped)
}

var localhostPortRe = regexp.MustCompile(`(?i)(?:localhost|127\.0\.0\.1|0\.0\.0\.0)[: ](\d{2,5})`)
var portFlagRe = regexp.MustCompile(`(?i)\bport[:= ]\s*(\d{2,5})\b`)

func extractPort(s string) int {
	if m := localhostPortRe.FindStringSubmatch(s); len(m) == 2 {
		if p, err := strconv.Atoi(m[1]); err == nil && p > 0 && p < 65536 {
			return p
		}
	}
	if m := portFlagRe.FindStringSubmatch(s); len(m) == 2 {
		if p, err := strconv.Atoi(m[1]); err == nil && p > 0 && p < 65536 {
			return p
		}
	}
	return 0
}

var readyMarkers = []string{
	"ready in",
	"ready - started",
	"local:",
	"localhost:",
	"listening on",
	"server running",
	"server started",
	"compiled successfully",
	"started development server",
}

func looksReady(s string) bool {
	low := strings.ToLower(s)
	for _, m := range readyMarkers {
		if strings.Contains(low, m) {
			return true
		}
	}
	return false
}
