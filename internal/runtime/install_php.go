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

	m.mu.RLock()
	phpAcq := m.phpAcquirer
	m.mu.RUnlock()
	if phpAcq == nil {
		return errors.New("composer install: no php acquirer configured")
	}
	phpEngine, ok := services.PHPEngineForVersion(proj.PHPVersion)
	if !ok {
		return fmt.Errorf("no bundled php engine for version %s", proj.PHPVersion)
	}
	phpFPMBin, err := phpAcq.Ensure(ctx, phpEngine)
	if err != nil {
		return fmt.Errorf("acquire php %s: %w", proj.PHPVersion, err)
	}
	phpBin := filepath.Join(filepath.Dir(phpFPMBin), "php")
	composerPhar, err := phpAcq.Ensure(ctx, services.ComposerEngine())
	if err != nil {
		return fmt.Errorf("acquire composer: %w", err)
	}

	sess.setPhase("install")
	m.recordSystem(proj.ID, LevelInfo, SourceSystem, "installing composer dependencies")
	m.emit(EvtStarting, StatusEvent{ProjectID: proj.ID, Snapshot: m.snap(sess)})

	cmd := exec.Command(phpBin, composerPhar, "install", "--no-interaction", "--no-progress")
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
