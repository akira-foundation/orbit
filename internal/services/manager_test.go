package services

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"syscall"
	"testing"
	"time"
)

type fakeRunner struct {
	mu         sync.Mutex
	lns        []net.Listener
	addr       string
	ignoreTerm bool
	lastProc   *svcProcess
}

func (f *fakeRunner) run(_ string, _, _ []string) (*svcProcess, error) {
	ln, err := net.Listen("tcp", f.addr)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	f.lns = append(f.lns, ln)
	ignoreTerm := f.ignoreTerm
	f.mu.Unlock()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	var cmd *exec.Cmd
	if ignoreTerm {
		cmd = exec.Command("sh", "-c", "trap '' TERM; while :; do sleep 1; done")
	} else {
		cmd = exec.Command("sleep", "60")
	}
	p, err := startCmdForTest(cmd)
	if err != nil {
		_ = ln.Close()
		return nil, err
	}
	f.mu.Lock()
	f.lastProc = p
	f.mu.Unlock()
	go func() {
		_ = p.cmd.Wait()
		_ = ln.Close()
	}()
	return p, nil
}

func (f *fakeRunner) closeAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, ln := range f.lns {
		_ = ln.Close()
	}
}

func startCmdForTest(cmd *exec.Cmd) (*svcProcess, error) {
	cmd.SysProcAttr = sysProcAttr()
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	pgid := cmd.Process.Pid
	return &svcProcess{cmd: cmd, pgid: pgid}, nil
}

func newManagerForTest(t *testing.T, webPort int) (*Manager, *fakeRunner) {
	t.Helper()
	store := newTestDB(t)
	acq := NewAcquirer(store, t.TempDir())
	resolver := NewResolver(store, AllowAll())
	m := NewManager(acq, resolver, LoadConfig(t.TempDir()), t.TempDir())

	fr := &fakeRunner{addr: fmt.Sprintf("127.0.0.1:%d", webPort)}
	m.starter = fr.run
	m.dialAddr = func(_ Engine) string { return fr.addr }
	m.owned = func(_ Engine) bool { return true }
	m.ensure = func(_ context.Context, _ Engine) (string, error) { return "fake-bin", nil }
	m.reap = func(string) {}
	m.reapForce = func(string) {}
	t.Cleanup(fr.closeAll)
	t.Cleanup(m.StopAll)
	return m, fr
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return p
}

func TestManagerRefcountStartsOnceStopsAtZero(t *testing.T) {
	port := freePort(t)
	m, _ := newManagerForTest(t, port)
	ctx := context.Background()

	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("acquire p1: %v", err)
	}
	if err := m.Acquire(ctx, "mailpit", "p2"); err != nil {
		t.Fatalf("acquire p2: %v", err)
	}
	if got := m.Status("mailpit"); got.Status != "running" || got.Refs != 2 {
		t.Fatalf("after 2 acquires: %+v", got)
	}

	m.Release("mailpit", "p1")
	if got := m.Status("mailpit"); got.Status != "running" || got.Refs != 1 {
		t.Fatalf("after 1 release: %+v", got)
	}

	m.Release("mailpit", "p2")
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status("mailpit").Status == "stopped" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := m.Status("mailpit"); got.Status != "stopped" || got.Refs != 0 {
		t.Fatalf("after all releases: %+v", got)
	}
}

func TestManagerAcquireUnknownEngine(t *testing.T) {
	m, _ := newManagerForTest(t, freePort(t))
	if err := m.Acquire(context.Background(), "ghost", "p1"); err == nil {
		t.Fatal("expected error for unknown engine")
	}
}

func TestAcquireRunsInitOnceBeforeStart(t *testing.T) {
	m, _ := newManagerForTest(t, freePort(t))
	var initCalls int
	var initBeforeStart bool
	var started bool
	origStarter := m.starter
	m.starter = func(bin string, args, env []string) (*svcProcess, error) {
		started = true
		return origStarter(bin, args, env)
	}
	m.initHook = func(e Engine, binDir, dataDir string) error {
		initCalls++
		if initCalls == 1 {
			initBeforeStart = !started
		}
		return nil
	}

	ctx := context.Background()
	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	m.Release("mailpit", "p1")
	time.Sleep(50 * time.Millisecond)
	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("re-acquire: %v", err)
	}

	if initCalls != 2 || !initBeforeStart {
		t.Fatalf("initCalls=%d beforeStart=%v", initCalls, initBeforeStart)
	}
}

func TestManagerUninstallStopsAndRemoves(t *testing.T) {
	m, _ := newManagerForTest(t, freePort(t))
	ctx := context.Background()

	if err := m.Acquire(ctx, "mailpit", "__manual__"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	_ = m.acq.store.RecordBinary(ctx, "mailpit", "v1.30.3", "/p/mailpit", "h")

	if err := m.Uninstall(ctx, "mailpit"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status("mailpit").Status == "stopped" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := m.Status("mailpit"); got.Status != "stopped" {
		t.Fatalf("expected stopped after uninstall, got %+v", got)
	}
	if _, ok, _ := m.acq.store.Binary(ctx, "mailpit", "v1.30.3"); ok {
		t.Fatal("expected binary row removed")
	}
}

func TestManagerForceStopClearsAllRefs(t *testing.T) {
	m, _ := newManagerForTest(t, freePort(t))
	ctx := context.Background()

	if err := m.Acquire(ctx, "mailpit", "__manual__"); err != nil {
		t.Fatalf("manual acquire: %v", err)
	}
	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("project acquire: %v", err)
	}
	if got := m.Status("mailpit"); got.Refs != 2 {
		t.Fatalf("expected 2 refs, got %+v", got)
	}

	m.ForceStop("mailpit")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status("mailpit").Status == "stopped" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := m.Status("mailpit"); got.Status != "stopped" || got.Refs != 0 {
		t.Fatalf("after force stop: %+v", got)
	}
}

func TestForceStopReapsUntrackedProcess(t *testing.T) {
	m, fr := newManagerForTest(t, freePort(t))
	ctx := context.Background()

	if err := m.Acquire(ctx, "mailpit", "p1"); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	fr.mu.Lock()
	proc := fr.lastProc
	fr.mu.Unlock()

	m.mu.Lock()
	delete(m.instances, "mailpit")
	m.mu.Unlock()

	m.reap = func(string) { _ = syscall.Kill(proc.pgid, syscall.SIGTERM) }
	m.reapForce = func(string) { _ = syscall.Kill(proc.pgid, syscall.SIGKILL) }

	if got := m.Status("mailpit"); got.Status != "running" {
		t.Fatalf("expected untracked process to read as running, got %+v", got)
	}

	m.ForceStop("mailpit")

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if m.Status("mailpit").Status == "stopped" {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("untracked process was not reaped, status=%+v", m.Status("mailpit"))
}

func TestTerminateEscalatesToSigkillWhenSigtermIgnored(t *testing.T) {
	m, fr := newManagerForTest(t, freePort(t))
	fr.mu.Lock()
	fr.ignoreTerm = true
	fr.mu.Unlock()

	if err := m.Acquire(context.Background(), "mailpit", "p1"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	m.ForceStop("mailpit")
	elapsed := time.Since(start)

	if got := m.Status("mailpit"); got.Status != "stopped" {
		t.Fatalf("expected stopped after escalation, got %+v", got)
	}
	if elapsed < 3*time.Second {
		t.Fatalf("expected escalation to wait out the grace period, took %s", elapsed)
	}
}
