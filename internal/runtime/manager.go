package runtime

import (
	"context"
	"database/sql"
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
	ConnOpen(projectID string)
	ConnClose(projectID string)
	Metrics(projectID string) []Sample
	MetricsAll() map[string][]Sample
	LogsHistory(projectID string, sinceTs int64, limit int) []LogLine
	RecordEvent(projectID string, level EventLevel, source EventSource, text string)
	RecordRequest(projectID string, statusCode int, durationMs float64, bytesIn, bytesOut int64, isWS bool)
	StopAll()
	Close() error
	SetEmitter(e Emitter)
}

func New(projects ProjectLookup, db *sql.DB) Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &manager{
		ctx:        ctx,
		cancel:     cancel,
		projects:   projects,
		emitter:    nopEmitter{},
		sessions:   make(map[string]*Session),
		stopGrace:  5 * time.Second,
		logCap:     2000,
		idleAfter:  10 * time.Second,
		activeConn: make(map[string]int),
		idleSince:  make(map[string]time.Time),
		metrics:    newMetrics(db),
		logs:       newLogStore(db),
	}
	go m.idleSweeper()
	go m.metricsSampler()
	return m
}

type manager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	projects ProjectLookup
	emitter  Emitter
	sessions map[string]*Session

	stopGrace time.Duration
	logCap    int

	idleAfter  time.Duration
	activeConn map[string]int
	idleSince  map[string]time.Time

	metrics *metrics
	logs    *logStore

	projMu      sync.Mutex
	projCounter map[string]*projMetrics
}

func (m *manager) projMetricsFor(projectID string) *projMetrics {
	m.projMu.Lock()
	defer m.projMu.Unlock()
	if m.projCounter == nil {
		m.projCounter = make(map[string]*projMetrics)
	}
	pm, ok := m.projCounter[projectID]
	if !ok {
		pm = &projMetrics{}
		m.projCounter[projectID] = pm
	}
	return pm
}

// RecordRequest is called by the proxy on every completed request. Increments
// counters used for req-rate, error-rate, latency percentiles, byte volume,
// HTTP-vs-WS breakdown.
func (m *manager) RecordRequest(projectID string, statusCode int, durationMs float64, bytesIn, bytesOut int64, isWS bool) {
	if projectID == "" {
		return
	}
	m.projMetricsFor(projectID).RecordRequest(statusCode, durationMs, bytesIn, bytesOut, isWS)
}

// addLog appends to the session's in-memory ring buffer (hot path for the
// streaming UI) and persists the line to the events table.
//
// Stream-based level inference: stderr -> warn, system -> info, stdout ->
// info. The events table is the single audit log for everything that
// happens in Orbit, kept forever by user request.
func (m *manager) addLog(sess *Session, line LogLine) {
	sess.appendLog(line)
	if m.logs == nil {
		return
	}
	level := LevelInfo
	if line.Stream == "stderr" {
		level = LevelWarn
	}
	m.logs.push(Event{
		Ts:        time.Now().UnixNano(),
		ProjectID: sess.projectID,
		SessionID: sess.sessionID,
		Level:     level,
		Source:    SourceRuntime,
		Stream:    line.Stream,
		Text:      line.Text,
	})
}

// recordSystem persists a non-runtime event (lifecycle marker, self-heal
// retry, crash detected, etc.). Project ID may be empty for app-wide events.
func (m *manager) recordSystem(projectID string, level EventLevel, source EventSource, text string) {
	if m.logs == nil {
		return
	}
	m.logs.push(Event{
		Ts:        time.Now().UnixNano(),
		ProjectID: projectID,
		Level:     level,
		Source:    source,
		Text:      text,
	})
}

func (m *manager) Metrics(projectID string) []Sample {
	return m.metrics.get(projectID)
}

func (m *manager) MetricsAll() map[string][]Sample {
	return m.metrics.all()
}

func (m *manager) LogsHistory(projectID string, sinceTs int64, limit int) []LogLine {
	if m.logs == nil {
		return nil
	}
	return m.logs.History(projectID, sinceTs, limit)
}

func (m *manager) RecordEvent(projectID string, level EventLevel, source EventSource, text string) {
	m.recordSystem(projectID, level, source, text)
}

// metricsSampler walks every active session every sampleInterval and pushes
// a snapshot point into the metric_samples table. Pulls in:
//   - session state (status, port, conns, uptime, attempts)
//   - traffic counters drained from projMetrics (reqs, errors, latency, bytes)
//   - OS process stats from ps (RSS, CPU%) for the child process group
//   - lifetime counters (crashes, autostops) and most recent wake latency
func (m *manager) metricsSampler() {
	tk := time.NewTicker(sampleInterval)
	defer tk.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-tk.C:
			m.mu.RLock()
			ids := make([]string, 0, len(m.sessions))
			pgids := make(map[string]int, len(m.sessions))
			for id, sess := range m.sessions {
				ids = append(ids, id)
				pgids[id] = sess.pgid
			}
			m.mu.RUnlock()
		now := time.Now().Unix()
		for _, id := range ids {
			snap := m.Status(id)
			d := m.projMetricsFor(id).Drain()
			ps := readProcStat(pgids[id])
			m.metrics.push(Sample{
				Ts:        now,
				ProjectID: id,
				Status:    snap.Status,
				Port:      snap.Port,
				Conns:     snap.Conns,
				UptimeMs:  snap.UptimeMs,
				Attempts:  snap.Attempts,
				ReqCount:  d.ReqCount,
				ErrCount:  d.ErrCount,
				HTTPReqs:  d.HTTPReqs,
				WSReqs:    d.WSReqs,
				BytesIn:   d.BytesIn,
				BytesOut:  d.BytesOut,
				P50Ms:     d.P50,
				P95Ms:     d.P95,
				P99Ms:     d.P99,
				MemKB:     ps.MemKB,
				CPUPct:    ps.CPUPct,
				Crashes:   d.Crashes,
				Autostops: d.Autostops,
				WakeMs:    d.WakeMs,
			})
		}
		}
	}
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

// snap takes a session and wraps the per-session Snapshot with manager-level
// metadata (currently the live connection count). Use this whenever you build
// a StatusEvent so emitted snapshots match what Status(id) returns.
func (m *manager) snap(sess *Session) Snapshot {
	s := sess.Snapshot()
	m.mu.RLock()
	s.Conns = m.activeConn[sess.projectID]
	m.mu.RUnlock()
	return s
}

func (m *manager) Start(ctx context.Context, projectID string) error {
	return m.start(ctx, projectID, false)
}

func (m *manager) start(ctx context.Context, projectID string, internal bool) error {
	m.mu.Lock()
	if existing, ok := m.sessions[projectID]; ok {
		st := existing.Status()
		if st == projects.StatusStarting || st == projects.StatusRunning {
			m.mu.Unlock()
			return errors.New("runtime: already running")
		}
		// External callers (proxy auto-wake, UI button) get throttled to
		// avoid relaunching on every browser refresh after a fresh failure.
		// Self-healing goroutines (internal=true) ignore this — they are
		// already paced by exponential backoff.
		if !internal && st == projects.StatusError && time.Since(existing.LastActivity()) < 10*time.Second {
			m.mu.Unlock()
			return errors.New("runtime: cooling down after recent failure")
		}
		// Defensive: kill any stale child group from the previous session
		// before spawning a new one so we never accumulate orphans.
		if existing.pgid > 0 {
			_ = killPGID(existing.pgid)
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
	m.recordSystem(proj.ID, LevelInfo, SourceSystem,
		fmt.Sprintf("starting runtime (cmd=%q port=%d)", proj.DevCommand, port))
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
	sess.pgid = handle.pgid
	// Don't pre-set sess.Port. The framework may ignore $PORT (Astro reads
	// astro.config, Vite reads vite.config) and bind a different one. Let
	// log parsing in readPipe resolve the actual listening port.

	_ = m.projects.UpdateStatus(ctx, projectID, projects.StatusStarting)
	m.emit(EvtStarting, StatusEvent{ProjectID: projectID, Snapshot: m.snap(sess)})

	go m.readPipe(sess, handle.stdout, "stdout")
	go m.readPipe(sess, handle.stderr, "stderr")
	go m.supervise(sess, handle, proj)
	go m.watchdog(sess, handle, 60*time.Second)

	return nil
}

// watchdog force-kills the process group if the session never reaches Running
// within timeout. Without this a hung dev server (compile loop, infinite
// retry, prompt waiting on stdin) would just sit there consuming RAM forever.
func (m *manager) watchdog(sess *Session, handle *processHandle, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-sess.stopCh:
		return
	case <-timer.C:
		if sess.Status() == projects.StatusStarting {
			log.Printf("[runtime] watchdog: project=%s never became ready in %s, killing pgid=%d",
				sess.projectID, timeout, handle.pgid)
			m.recordSystem(sess.projectID, LevelError, SourceSystem,
				fmt.Sprintf("watchdog killed runtime: never became ready in %s", timeout))
			_ = handle.forceKill()
		}
	}
}

func (m *manager) markStartFailed(sess *Session, err error) {
	sess.setError(err.Error())
	m.emit(EvtError, StatusEvent{ProjectID: sess.projectID, Snapshot: m.snap(sess)})
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
		s := sess.Snapshot()
		s.Conns = m.activeConn[projectID]
		return s
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

// ConnOpen and ConnClose are called by the proxy on every incoming request
// (including long-lived WebSocket / SSE connections). When a project's active
// connection count drops to zero, idleSweeper schedules a graceful stop after
// idleAfter so closing the browser tab actually shuts the dev server down.
func (m *manager) ConnOpen(projectID string) {
	if projectID == "" {
		return
	}
	m.mu.Lock()
	m.activeConn[projectID]++
	delete(m.idleSince, projectID)
	m.mu.Unlock()
}

func (m *manager) ConnClose(projectID string) {
	if projectID == "" {
		return
	}
	m.mu.Lock()
	if n := m.activeConn[projectID]; n > 1 {
		m.activeConn[projectID] = n - 1
	} else {
		delete(m.activeConn, projectID)
		m.idleSince[projectID] = time.Now()
	}
	m.mu.Unlock()
}

func (m *manager) idleSweeper() {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-tick.C:
			m.mu.RLock()
			var toStop []string
			now := time.Now()
			for id, since := range m.idleSince {
				if now.Sub(since) < m.idleAfter {
					continue
				}
				if sess, ok := m.sessions[id]; ok {
					st := sess.Status()
					if st == projects.StatusRunning || st == projects.StatusStarting {
						toStop = append(toStop, id)
					}
				}
			}
			m.mu.RUnlock()

		for _, id := range toStop {
			log.Printf("[runtime] idle stop project=%s after %s", id, m.idleAfter)
			m.recordSystem(id, LevelInfo, SourceSystem,
				fmt.Sprintf("idle auto-stop after %s with no active connections", m.idleAfter))
			m.projMetricsFor(id).IncAutostop()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = m.Stop(ctx, id)
			cancel()
			m.mu.Lock()
			delete(m.idleSince, id)
			m.mu.Unlock()
		}
		}
	}
}

func (m *manager) StopAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	pgids := make([]int, 0, len(m.sessions))
	for id, s := range m.sessions {
		st := s.Status()
		if st == projects.StatusStarting || st == projects.StatusRunning {
			ids = append(ids, id)
		}
		if s.pgid > 0 {
			pgids = append(pgids, s.pgid)
		}
	}
	m.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	for _, id := range ids {
		_ = m.Stop(ctx, id)
	}
	// Final sweep: regardless of graceful stop, force-kill every known
	// process group so app exit never leaves dev servers running.
	for _, pgid := range pgids {
		_ = killPGID(pgid)
	}
}

func (m *manager) Close() error {
	m.StopAll()
	if m.cancel != nil {
		m.cancel()
	}
	return nil
}

func (m *manager) readPipe(sess *Session, r io.Reader, stream string) {
	scanLines(r, func(text string) bool {
		line := LogLine{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Stream:    stream,
			Text:      text,
		}
		m.addLog(sess, line)
		m.emit(EvtLog, LogEvent{ProjectID: sess.projectID, Line: line})

		if port, strong := extractPortFromLine(text); port > 0 {
			if strong || sess.Port() == 0 {
				sess.setPort(port)
			}
		}
		if sess.Status() == projects.StatusStarting && looksReady(text) {
			sess.setStatus(projects.StatusRunning)
			m.projMetricsFor(sess.projectID).SetWakeMs(sinceMs(sess.startedAt))
			_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusRunning)
			m.emit(EvtRunning, StatusEvent{ProjectID: sess.projectID, Snapshot: m.snap(sess)})
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
			select {
			case err := <-doneCh:
				m.finalize(sess, err, true)
			case <-time.After(time.Second):
				m.finalize(sess, errors.New("killed forcefully"), true)
			}
		}
	}
}

func (m *manager) finalize(sess *Session, err error, requested bool) {
	if requested {
		sess.markStopped()
		m.emit(EvtStopped, StatusEvent{ProjectID: sess.projectID, Snapshot: m.snap(sess)})
		_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusStopped)
		return
	}

	prematurelyExited := sess.Status() == projects.StatusStarting
	wasRunning := sess.WasRunning()

	if err != nil || prematurelyExited {
		msg := "process exited before becoming ready"
		if err != nil {
			msg = "process exited: " + err.Error()
		}
		sess.setError(msg)
		m.addLog(sess, LogLine{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Stream:    "system",
			Text:      msg,
		})
		m.emit(EvtError, StatusEvent{ProjectID: sess.projectID, Snapshot: m.snap(sess)})
		_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusError)
		if wasRunning {
			// Treat as a real crash for metrics purposes — startups that
			// never reached Running are config bugs, not crashes.
			m.projMetricsFor(sess.projectID).IncCrash()
		}
		// Self-healing: only retry runtimes that previously reached Running.
		// A startup that never went healthy is most likely a config bug
		// (wrong port, missing dep) and retrying just spams the user.
		if wasRunning {
			m.scheduleRestart(sess)
		}
		return
	}

	// Clean exit (code 0) of a previously-running session: still try to
	// recover — the user did not request a stop.
	sess.markStopped()
	m.emit(EvtStopped, StatusEvent{ProjectID: sess.projectID, Snapshot: m.snap(sess)})
	_ = m.projects.UpdateStatus(context.Background(), sess.projectID, projects.StatusStopped)
	if wasRunning {
		m.scheduleRestart(sess)
	}
}

// scheduleRestart kicks off an exponential-backoff retry after an unexpected
// crash. Each attempt doubles the wait, capped at maxBackoff. Gives up after
// maxRestartAttempts and leaves the session in Error so the recovery overlay
// stays visible.
const (
	maxRestartAttempts = 5
	maxBackoff         = 16 * time.Second
)

func (m *manager) scheduleRestart(sess *Session) {
	n := sess.bumpAttempts()
	if n > maxRestartAttempts {
		log.Printf("[runtime] self-heal giving up project=%s attempts=%d",
			sess.projectID, n)
		m.recordSystem(sess.projectID, LevelError, SourceSystem,
			fmt.Sprintf("self-heal giving up after %d attempts", n))
		return
	}
	delay := time.Duration(1<<uint(n-1)) * time.Second
	if delay > maxBackoff {
		delay = maxBackoff
	}
	log.Printf("[runtime] self-heal scheduling project=%s attempt=%d in=%s",
		sess.projectID, n, delay)
	m.recordSystem(sess.projectID, LevelWarn, SourceSystem,
		fmt.Sprintf("self-heal restart attempt %d scheduled in %s", n, delay))
	go func() {
		t := time.NewTimer(delay)
		defer t.Stop()
		select {
		case <-sess.stopCh:
			return
		case <-t.C:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := m.start(ctx, sess.projectID, true); err != nil {
			log.Printf("[runtime] self-heal restart failed project=%s err=%v",
				sess.projectID, err)
		}
	}()
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

var localhostPortRe = regexp.MustCompile(`(?i)(?:localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1\])[: ](\d{2,5})`)
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
