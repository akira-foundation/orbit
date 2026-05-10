package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
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
	Port(projectID string) int
	IsRunning(projectID string) bool
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
		if st == projects.StatusError && time.Since(existing.LastActivity()) < 10*time.Second {
			m.mu.Unlock()
			return errors.New("runtime: cooling down after recent failure")
		}
	}
	sess := newSession(projectID, m.logCap)
	sess.setStatus(projects.StatusStarting)
	m.sessions[projectID] = sess
	m.mu.Unlock()

	proj, err := m.projects.Get(ctx, projectID)
	if err != nil {
		m.markStartFailed(sess, err)
		return fmt.Errorf("runtime: load project: %w", err)
	}
	if proj.DevCommand == "" {
		err := errors.New("runtime: project has no dev command")
		m.markStartFailed(sess, err)
		return err
	}

	port := pickPort(proj.DevPort)
	log.Printf("[runtime] start project=%s devPort=%d picked=%d cmd=%q",
		proj.ID, proj.DevPort, port, proj.DevCommand)
	env := append(os.Environ(),
		"FORCE_COLOR=1",
		"CI=false",
		fmt.Sprintf("PORT=%d", port),
	)

	handle, err := spawnDevCommand(proj.Path, proj.DevCommand, env)
	if err != nil {
		m.markStartFailed(sess, err)
		return fmt.Errorf("runtime: spawn: %w", err)
	}

	sess.setPID(handle.cmd.Process.Pid)
	sess.setPort(port)

	_ = m.projects.UpdateStatus(ctx, projectID, projects.StatusStarting)
	m.emit(EvtStarting, StatusEvent{ProjectID: projectID, Snapshot: sess.Snapshot()})

	go m.readPipe(sess, handle.stdout, "stdout")
	go m.readPipe(sess, handle.stderr, "stderr")
	go m.supervise(sess, handle, proj)

	return nil
}

func (m *manager) markStartFailed(sess *Session, err error) {
	sess.setError(err.Error())
	m.emit(EvtError, StatusEvent{ProjectID: sess.projectID, Snapshot: sess.Snapshot()})
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

func (m *manager) Port(projectID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[projectID]; ok {
		return sess.Port()
	}
	return 0
}

func (m *manager) IsRunning(projectID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if sess, ok := m.sessions[projectID]; ok {
		st := sess.Status()
		return st == projects.StatusRunning || st == projects.StatusStarting
	}
	return false
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

		if port, strong := extractPortFromLine(text); port > 0 {
			if strong || sess.Port() == 0 {
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

	// Process exited on its own. If it never reached Running, treat as a
	// failure so the cooldown kicks in and we don't relaunch on every request.
	prematurelyExited := sess.Status() == projects.StatusStarting

	if err != nil || prematurelyExited {
		msg := "process exited before becoming ready"
		if err != nil {
			msg = "process exited: " + err.Error()
		}
		sess.setError(msg)
		sess.appendLog(LogLine{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Stream:    "system",
			Text:      msg,
		})
		m.emit(EvtError, StatusEvent{ProjectID: sess.projectID, Snapshot: sess.Snapshot()})
		_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusError)
		return
	}
	sess.markStopped()
	m.emit(EvtStopped, StatusEvent{ProjectID: sess.projectID, Snapshot: sess.Snapshot()})
	_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusStopped)
}

// pickPort returns a free ephemeral port for the dev server. We always pick
// fresh — the proxy hides port numbers from the user, and ephemeral avoids
// races / collisions between multiple Orbit projects all defaulting to 3000.
// preferred is only used as a fallback if the OS refuses to allocate.
func pickPort(preferred int) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if preferred > 0 {
			return preferred
		}
		return 3000
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

var localhostPortRe = regexp.MustCompile(`(?i)(?:localhost|127\.0\.0\.1|0\.0\.0\.0)[: ](\d{2,5})`)
var portFlagRe = regexp.MustCompile(`(?i)\bport[:= ]\s*(\d{2,5})\b`)

// extractPortFromLine returns (port, strong). A "strong" match is one tied to
// an actual listener address (localhost:N / 127.0.0.1:N / 0.0.0.0:N) and should
// overwrite previously detected ports. Weak matches (e.g. "Port 3000 is in use")
// only set the port if none was found yet.
func extractPortFromLine(s string) (int, bool) {
	if m := localhostPortRe.FindStringSubmatch(s); len(m) == 2 {
		if p, err := strconv.Atoi(m[1]); err == nil && p > 0 && p < 65536 {
			return p, true
		}
	}
	if m := portFlagRe.FindStringSubmatch(s); len(m) == 2 {
		if p, err := strconv.Atoi(m[1]); err == nil && p > 0 && p < 65536 {
			return p, false
		}
	}
	return 0, false
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
