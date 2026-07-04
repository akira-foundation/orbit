package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"orbit-app/internal/projects"
	"orbit-app/internal/services"
)

// prefers a global `composer`, then system php + bundled composer.phar, then fully bundled
func (m *manager) resolveComposerCommand(ctx context.Context, proj *projects.Project) ([]string, error) {
	m.mu.RLock()
	rtCfg := m.runtimesConfig
	phpAcq := m.phpAcquirer
	m.mu.RUnlock()

	preferSystem := rtCfg != nil && rtCfg.PreferSystemPHP()
	sys, sysOK := services.DetectSystemPHP()

	if preferSystem && sysOK {
		if composerPath, err := exec.LookPath("composer"); err == nil {
			return []string{composerPath}, nil
		}
	}

	if phpAcq == nil {
		return nil, errors.New("composer install: no php acquirer configured")
	}
	composerPhar, err := phpAcq.Ensure(ctx, services.ComposerEngine())
	if err != nil {
		return nil, fmt.Errorf("acquire composer: %w", err)
	}

	if preferSystem && sysOK {
		return []string{sys.PHPPath, composerPhar}, nil
	}

	phpEngine, ok := services.PHPEngineForVersion(proj.PHPVersion)
	if !ok {
		return nil, fmt.Errorf("no bundled php engine for version %s", proj.PHPVersion)
	}
	phpFPMBin, err := phpAcq.Ensure(ctx, phpEngine)
	if err != nil {
		return nil, fmt.Errorf("acquire php %s: %w", proj.PHPVersion, err)
	}
	phpBin := filepath.Join(filepath.Dir(phpFPMBin), "php")
	return []string{phpBin, composerPhar}, nil
}

func composerLockHash(dir string) (string, error) {
	b, err := os.ReadFile(filepath.Join(dir, "composer.lock"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}

func needsComposerInstall(proj *projects.Project, knownHash string) bool {
	if proj.Path == "" {
		return false
	}
	vendor := filepath.Join(proj.Path, "vendor")
	if st, err := os.Stat(vendor); err != nil || !st.IsDir() {
		return true
	}
	cur, _ := composerLockHash(proj.Path)
	if cur == "" {
		return false
	}
	return cur != knownHash
}

func (m *manager) ensureComposerInstalled(ctx context.Context, sess *Session, proj *projects.Project) error {
	knownHash, _ := m.projects.InstalledHash(ctx, proj.ID)
	if !needsComposerInstall(proj, knownHash) {
		return nil
	}

	argv, err := m.resolveComposerCommand(ctx, proj)
	if err != nil {
		return err
	}

	sess.setPhase("install")
	m.recordSystem(proj.ID, LevelInfo, SourceSystem, "installing composer dependencies")
	m.emit(EvtStarting, StatusEvent{ProjectID: proj.ID, Snapshot: m.snap(sess)})

	cmd := exec.Command(argv[0], append(argv[1:], "install", "--no-interaction", "--no-progress")...)
	cmd.Dir = proj.Path
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1", "CI=false")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	pgid, perr := syscall.Getpgid(cmd.Process.Pid)
	if perr != nil {
		pgid = cmd.Process.Pid
	}
	h := &processHandle{cmd: cmd, stdout: stdout, stderr: stderr, pgid: pgid}

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
		return errors.New("composer install: cancelled")
	case err := <-done:
		if err != nil {
			return err
		}
	}

	hash, _ := composerLockHash(proj.Path)
	if hash != "" {
		if err := m.projects.SetInstalledHash(ctx, proj.ID, hash); err != nil {
			log.Printf("[runtime] save installed hash: %v", err)
		}
	}
	m.recordSystem(proj.ID, LevelInfo, SourceSystem, "composer install complete")
	sess.setPhase("")
	return nil
}
