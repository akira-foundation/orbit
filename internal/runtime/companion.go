package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"orbit-app/internal/projects"
)

func frontendProcessSpec(proj *projects.Project) *projects.ProcessSpec {
	for i := range proj.Processes {
		if proj.Processes[i].Role == projects.ProcessRoleFrontend {
			return &proj.Processes[i]
		}
	}
	return nil
}

const hotFileTimeout = 30 * time.Second

func (m *manager) startCompanionAndWait(ctx context.Context, sess *Session, proj *projects.Project) {
	spec := frontendProcessSpec(proj)
	if spec == nil {
		return
	}
	m.startCompanion(ctx, sess, proj)

	docRoot := filepath.Join(proj.Path, "public")
	if err := waitHotFile(ctx, docRoot, hotFileTimeout); err != nil {
		m.recordSystem(proj.ID, LevelWarn, SourceSystem, fmt.Sprintf("frontend companion: %v", err))
	}
}

func waitHotFile(ctx context.Context, docRoot string, timeout time.Duration) error {
	hotPath := filepath.Join(docRoot, "hot")
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if st, err := os.Stat(hotPath); err == nil && !st.IsDir() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return fmt.Errorf("vite dev server never wrote %s within %s", hotPath, timeout)
}

func (m *manager) startCompanion(ctx context.Context, sess *Session, proj *projects.Project) {
	spec := frontendProcessSpec(proj)
	if spec == nil {
		return
	}
	dir := filepath.Join(proj.Path, spec.WorkDir)

	nodeBinDir, err := m.nodeBinDir(ctx, proj)
	if err != nil {
		m.recordSystem(proj.ID, LevelWarn, SourceSystem, fmt.Sprintf("frontend companion: node runtime: %v", err))
		return
	}

	if needsNodeInstall(dir) {
		pm := detectCompanionPM(dir)
		if err := runCompanionInstall(dir, pm, nodeBinDir); err != nil {
			m.recordSystem(proj.ID, LevelWarn, SourceSystem, fmt.Sprintf("frontend companion: install: %v", err))
			return
		}
	}

	env := append(stripEnvVar(os.Environ(), "CI"), "FORCE_COLOR=1", "ORBIT_MANAGED=1")
	env = mergeDotEnv(env, dir)
	if proj.LocalDomain != "" {
		env = append(env, "APP_URL=https://"+proj.LocalDomain)
	}
	env = prependNodeBinDir(env, nodeBinDir)

	handle, err := spawnDevCommand(dir, spec.Command, env)
	if err != nil {
		m.recordSystem(proj.ID, LevelWarn, SourceSystem, fmt.Sprintf("frontend companion: spawn: %v", err))
		return
	}

	sess.setCompanionPID(handle.cmd.Process.Pid, handle.pgid)
	sess.companionKillFn = func() { _ = handle.kill() }

	go m.readCompanionPipe(sess, handle.stdout, "stdout")
	go m.readCompanionPipe(sess, handle.stderr, "stderr")
	go m.companionSupervise(sess, handle)

	m.recordSystem(proj.ID, LevelInfo, SourceSystem,
		fmt.Sprintf("frontend companion started (cmd=%q dir=%q)", spec.Command, dir))
}

func (m *manager) readCompanionPipe(sess *Session, r io.Reader, stream string) {
	scanLines(r, func(text string) bool {
		line := LogLine{
			Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
			Stream:    stream,
			Text:      "[frontend] " + text,
		}
		m.addLog(sess, line)
		m.emit(EvtLog, LogEvent{ProjectID: sess.projectID, Line: line})
		return true
	})
}

func (m *manager) companionSupervise(sess *Session, handle *processHandle) {
	doneCh := make(chan error, 1)
	go func() { doneCh <- handle.cmd.Wait() }()

	select {
	case err := <-doneCh:
		if err != nil {
			m.recordSystem(sess.projectID, LevelWarn, SourceSystem, fmt.Sprintf("frontend companion exited: %v", err))
		}
	case <-sess.stopCh:
		select {
		case <-doneCh:
		case <-time.After(m.stopGrace):
			_ = handle.forceKill()
			<-doneCh
		}
	}
}

func needsNodeInstall(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "node_modules"))
	return err != nil || !st.IsDir()
}

func detectCompanionPM(dir string) string {
	checks := []struct{ file, pm string }{
		{"bun.lockb", "bun"},
		{"bun.lock", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			return c.pm
		}
	}
	return "npm"
}

func runCompanionInstall(dir, pm, nodeBinDir string) error {
	cmd := installCommand(pm)
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Dir = dir
	env := append(stripEnvVar(os.Environ(), "CI"), "FORCE_COLOR=1")
	c.Env = prependNodeBinDir(env, nodeBinDir)
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return c.Run()
}
