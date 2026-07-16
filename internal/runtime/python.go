package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"orbit-app/internal/projects"
	"orbit-app/internal/services"
)

func (m *manager) SetPythonAcquirer(a *services.Acquirer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pythonAcquirer = a
}

func (m *manager) pythonBinDir(ctx context.Context, proj *projects.Project) (string, error) {
	if proj.PythonVersion == "" {
		return "", nil
	}
	interpreter, err := m.pythonInterpreter(ctx, proj)
	if err != nil {
		return "", err
	}
	venvBin, err := ensureVenv(proj.Path, interpreter)
	if err != nil {
		return "", fmt.Errorf("create venv: %w", err)
	}
	return venvBin, nil
}

func (m *manager) pythonInterpreter(ctx context.Context, proj *projects.Project) (string, error) {
	m.mu.RLock()
	acq := m.pythonAcquirer
	rtCfg := m.runtimesConfig
	m.mu.RUnlock()

	if rtCfg != nil && rtCfg.PreferSystemPython() {
		if sys, ok := services.DetectSystemPython(); ok {
			return filepath.Join(sys.BinDir, "python3"), nil
		}
	}
	if acq == nil {
		return "", fmt.Errorf("no python acquirer configured")
	}
	engine, ok := services.PythonEngineForVersion(proj.PythonVersion)
	if !ok {
		return "", fmt.Errorf("no bundled python engine for version %s", proj.PythonVersion)
	}
	binPath, err := acq.Ensure(ctx, engine)
	if err != nil {
		return "", fmt.Errorf("acquire python %s: %w", proj.PythonVersion, err)
	}
	return binPath, nil
}

func ensureVenv(projectPath, interpreter string) (string, error) {
	venvDir := filepath.Join(projectPath, ".venv")
	venvPython := filepath.Join(venvDir, "bin", "python")
	if st, err := os.Stat(venvPython); err == nil && !st.IsDir() {
		return filepath.Join(venvDir, "bin"), nil
	}
	cmd := exec.Command(interpreter, "-m", "venv", venvDir)
	cmd.Dir = projectPath
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%w: %s", err, out)
	}
	return filepath.Join(venvDir, "bin"), nil
}

var pythonLockfiles = []string{"uv.lock", "poetry.lock", "Pipfile.lock", "requirements.txt", "pyproject.toml"}

func pythonLockHash(dir string) string {
	for _, name := range pythonLockfiles {
		p := filepath.Join(dir, name)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		h := sha256.Sum256(b)
		return hex.EncodeToString(h[:])
	}
	return ""
}

func needsPythonInstall(proj *projects.Project, knownHash string) bool {
	if proj.Path == "" {
		return false
	}
	venvPython := filepath.Join(proj.Path, ".venv", "bin", "python")
	if st, err := os.Stat(venvPython); err != nil || st.IsDir() {
		return true
	}
	cur := pythonLockHash(proj.Path)
	if cur == "" {
		return false
	}
	return cur != knownHash
}

func pythonInstallCommand(proj *projects.Project) []string {
	hasFile := func(name string) bool {
		_, err := os.Stat(filepath.Join(proj.Path, name))
		return err == nil
	}
	switch proj.PackageManager {
	case "uv":
		if _, err := exec.LookPath("uv"); err == nil {
			return []string{"uv", "sync"}
		}
	case "poetry":
		if _, err := exec.LookPath("poetry"); err == nil {
			return []string{"poetry", "install"}
		}
	case "pipenv":
		if _, err := exec.LookPath("pipenv"); err == nil {
			return []string{"pipenv", "install"}
		}
	}
	if hasFile("requirements.txt") {
		return []string{"python", "-m", "pip", "install", "-r", "requirements.txt"}
	}
	if hasFile("pyproject.toml") {
		return []string{"python", "-m", "pip", "install", "."}
	}
	return nil
}

func resolvePythonBin(bin, venvBin string) string {
	if bin != "python" || venvBin == "" {
		return bin
	}
	venvPython := filepath.Join(venvBin, "python")
	if st, err := os.Stat(venvPython); err == nil && !st.IsDir() {
		return venvPython
	}
	return bin
}

func pythonInstallHandle(proj *projects.Project, venvBin string) (*processHandle, error) {
	cmd := pythonInstallCommand(proj)
	if len(cmd) == 0 {
		return nil, nil
	}
	env := append(stripEnvVar(os.Environ(), "CI"),
		"VIRTUAL_ENV="+filepath.Join(proj.Path, ".venv"),
		"POETRY_VIRTUALENVS_IN_PROJECT=1",
		"PIPENV_VENV_IN_PROJECT=1",
		"UV_PROJECT_ENVIRONMENT="+filepath.Join(proj.Path, ".venv"),
	)
	env = withLoginPath(prependNodeBinDir(env, venvBin))
	bin := resolvePythonBin(cmd[0], venvBin)
	if bin == cmd[0] {
		bin = lookPathIn(cmd[0], envPathValue(env))
	}
	c := exec.Command(bin, cmd[1:]...)
	c.Dir = proj.Path
	c.Env = env
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := c.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := c.Start(); err != nil {
		return nil, err
	}
	pgid, err := syscall.Getpgid(c.Process.Pid)
	if err != nil {
		pgid = c.Process.Pid
	}
	return &processHandle{cmd: c, stdout: stdout, stderr: stderr, pgid: pgid}, nil
}

func (m *manager) runPythonInstall(ctx context.Context, sess *Session, proj *projects.Project, venvBin string) error {
	sess.setPhase("install")
	cmd := pythonInstallCommand(proj)
	if len(cmd) == 0 {
		sess.setPhase("")
		return nil
	}
	m.recordSystem(proj.ID, LevelInfo, SourceSystem,
		fmt.Sprintf("installing dependencies (%s)", joinArgs(cmd)))
	m.emit(EvtStarting, StatusEvent{ProjectID: proj.ID, Snapshot: m.snap(sess)})

	h, err := pythonInstallHandle(proj, venvBin)
	if err != nil {
		return err
	}
	if h == nil {
		sess.setPhase("")
		return nil
	}
	sess.setPID(h.cmd.Process.Pid)
	sess.pgid = h.pgid

	done := make(chan error, 1)
	go m.readPipe(sess, h.stdout, "stdout")
	go m.readPipe(sess, h.stderr, "stderr")
	go func() { done <- h.cmd.Wait() }()

	select {
	case <-sess.stopCh:
		_ = h.forceKill()
		<-done
		return fmt.Errorf("install: cancelled")
	case err := <-done:
		if err != nil {
			return err
		}
	}

	if hash := pythonLockHash(proj.Path); hash != "" {
		_ = m.projects.SetInstalledHash(ctx, proj.ID, hash)
	}
	m.recordSystem(proj.ID, LevelInfo, SourceSystem, "install complete")
	sess.setPhase("")
	return nil
}

func substitutePort(cmd string, port int) string {
	p := strconv.Itoa(port)
	cmd = strings.ReplaceAll(cmd, "${PORT}", p)
	return strings.ReplaceAll(cmd, "$PORT", p)
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}
