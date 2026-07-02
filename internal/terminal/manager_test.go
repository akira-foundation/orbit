package terminal

import (
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeEmitter struct {
	mu     sync.Mutex
	events map[string][]any
}

func newFakeEmitter() *fakeEmitter {
	return &fakeEmitter{events: map[string][]any{}}
}

func (f *fakeEmitter) Emit(name string, data ...any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events[name] = append(f.events[name], data...)
}

func (f *fakeEmitter) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.events[name])
}

func waitForBuffer(t *testing.T, m *Manager, projectID, want string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(string(m.Buffer(projectID)), want) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("buffer never contained %q, got %q", want, string(m.Buffer(projectID)))
}

func TestManagerStartWriteRoundTrip(t *testing.T) {
	m := NewManager()
	m.SetEmitter(newFakeEmitter())
	t.Cleanup(m.StopAll)

	dir := t.TempDir()
	if err := m.Start("p1", dir); err != nil {
		t.Fatalf("start: %v", err)
	}

	if err := m.Write("p1", []byte("echo hello-terminal\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	waitForBuffer(t, m, "p1", "hello-terminal", 3*time.Second)
}

func TestManagerStartIsIdempotent(t *testing.T) {
	m := NewManager()
	m.SetEmitter(newFakeEmitter())
	t.Cleanup(m.StopAll)

	dir := t.TempDir()
	if err := m.Start("p1", dir); err != nil {
		t.Fatalf("first start: %v", err)
	}
	pid1 := m.sessions["p1"].cmd.Process.Pid

	if err := m.Start("p1", dir); err != nil {
		t.Fatalf("second start: %v", err)
	}
	pid2 := m.sessions["p1"].cmd.Process.Pid

	if pid1 != pid2 {
		t.Fatalf("expected same pid, got %d then %d", pid1, pid2)
	}
}

func TestManagerStopKillsProcess(t *testing.T) {
	m := NewManager()
	m.SetEmitter(newFakeEmitter())

	dir := t.TempDir()
	if err := m.Start("p1", dir); err != nil {
		t.Fatalf("start: %v", err)
	}
	proc := m.sessions["p1"].cmd.Process

	m.Stop("p1")

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if proc.Signal(nil) != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("process still alive after Stop")
}

func TestManagerResizeDoesNotError(t *testing.T) {
	m := NewManager()
	m.SetEmitter(newFakeEmitter())
	t.Cleanup(m.StopAll)

	dir := t.TempDir()
	if err := m.Start("p1", dir); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := m.Resize("p1", 120, 40); err != nil {
		t.Fatalf("resize: %v", err)
	}
}

func TestManagerEmitsDataEvents(t *testing.T) {
	m := NewManager()
	fe := newFakeEmitter()
	m.SetEmitter(fe)
	t.Cleanup(m.StopAll)

	dir := t.TempDir()
	if err := m.Start("p1", dir); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := m.Write("p1", []byte("echo hi\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if fe.count(DataEventName("p1")) > 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("no data event emitted")
}
