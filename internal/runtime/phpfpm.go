package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"orbit-app/internal/projects"
	"orbit-app/internal/services"
)

func (m *manager) SetPHPAcquirer(a *services.Acquirer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phpAcquirer = a
}

func (m *manager) SocketPath(projectID string) string {
	m.mu.RLock()
	sess, ok := m.sessions[projectID]
	m.mu.RUnlock()
	if !ok {
		return ""
	}
	return sess.SockPath()
}

// fixed-length hash, not the raw slug: sockaddr_un caps socket paths around 104 bytes
func phpRunID(projectID string) string {
	sum := sha256.Sum256([]byte(projectID))
	return hex.EncodeToString(sum[:])[:12]
}

func (m *manager) phpRunDir(projectID string) string {
	return filepath.Join(m.dataDir, "php-run", phpRunID(projectID))
}

func (m *manager) startPHP(ctx context.Context, sess *Session, proj *projects.Project) error {
	if err := m.ensureComposerInstalled(ctx, sess, proj); err != nil {
		m.markStartFailed(sess, err)
		return fmt.Errorf("runtime: composer install: %w", err)
	}

	m.servicesOnStart(proj)

	handle, sockPath, err := m.spawnPHPFPM(ctx, proj)
	if err != nil {
		m.markStartFailed(sess, err)
		return fmt.Errorf("runtime: php-fpm: %w", err)
	}

	sess.setPID(handle.cmd.Process.Pid)
	sess.pgid = handle.pgid
	sess.setSockPath(sockPath)

	_ = m.projects.UpdateStatus(ctx, proj.ID, projects.StatusStarting)
	m.emit(EvtStarting, StatusEvent{ProjectID: proj.ID, Snapshot: m.snap(sess)})

	go m.readPipe(sess, handle.stdout, "stdout")
	go m.readPipe(sess, handle.stderr, "stderr")
	go m.supervise(sess, handle, proj)
	go m.watchdog(sess, handle, 60*time.Second)

	if err := waitUnixSocket(ctx, sockPath, 15*time.Second); err != nil {
		_ = handle.forceKill()
		m.markStartFailed(sess, err)
		return fmt.Errorf("runtime: php-fpm: %w", err)
	}
	sess.setStatus(projects.StatusRunning)
	m.projMetricsFor(proj.ID).SetWakeMs(sinceMs(sess.startedAt))
	_ = m.projects.UpdateStatus(ctx, proj.ID, projects.StatusRunning)
	m.emit(EvtRunning, StatusEvent{ProjectID: proj.ID, Snapshot: m.snap(sess)})
	return nil
}

// system php-fpm wins when preferred and detected, else the bundled PHPVersion build
func (m *manager) resolvePHPFPMBin(ctx context.Context, proj *projects.Project) (string, error) {
	m.mu.RLock()
	rtCfg := m.runtimesConfig
	acq := m.phpAcquirer
	m.mu.RUnlock()

	if rtCfg != nil && rtCfg.PreferSystemPHP() {
		if sys, ok := services.DetectSystemPHP(); ok {
			return sys.PHPFPMPath, nil
		}
	}

	if acq == nil {
		return "", errors.New("php-fpm: no acquirer configured")
	}
	engine, ok := services.PHPEngineForVersion(proj.PHPVersion)
	if !ok {
		return "", fmt.Errorf("no bundled php engine for version %s", proj.PHPVersion)
	}
	binPath, err := acq.Ensure(ctx, engine)
	if err != nil {
		return "", fmt.Errorf("acquire php %s: %w", proj.PHPVersion, err)
	}
	return binPath, nil
}

func (m *manager) spawnPHPFPM(ctx context.Context, proj *projects.Project) (*processHandle, string, error) {
	fpmBin, err := m.resolvePHPFPMBin(ctx, proj)
	if err != nil {
		return nil, "", err
	}

	runDir := m.phpRunDir(proj.ID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, "", err
	}
	sockPath := filepath.Join(runDir, "f.sock")
	docRoot := filepath.Join(proj.Path, "public")
	confPath, err := writePoolConfig(runDir, sockPath, docRoot)
	if err != nil {
		return nil, "", err
	}

	cmd := exec.Command(fpmBin, "-n", "-y", confPath, "-F", "-O")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, "", fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, "", fmt.Errorf("stderr pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, "", fmt.Errorf("start php-fpm: %w", err)
	}
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		pgid = cmd.Process.Pid
	}
	return &processHandle{cmd: cmd, stdout: stdout, stderr: stderr, pgid: pgid}, sockPath, nil
}

func writePoolConfig(runDir, sockPath, docRoot string) (string, error) {
	conf := filepath.Join(runDir, "pool.conf")
	body := "[global]\n" +
		"pid = " + filepath.Join(runDir, "fpm.pid") + "\n" +
		"error_log = " + filepath.Join(runDir, "error.log") + "\n" +
		"daemonize = no\n\n" +
		"[orbit]\n" +
		"listen = " + sockPath + "\n" +
		"pm = ondemand\n" +
		"pm.max_children = 5\n" +
		"pm.process_idle_timeout = 10s\n" +
		"pm.max_requests = 500\n" +
		"catch_workers_output = yes\n" +
		"clear_env = no\n" +
		"chdir = " + docRoot + "\n"
	if err := os.WriteFile(conf, []byte(body), 0o644); err != nil {
		return "", err
	}
	return conf, nil
}

func waitUnixSocket(ctx context.Context, sockPath string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if conn, err := net.Dial("unix", sockPath); err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("php-fpm: socket %s never became dialable", sockPath)
}
