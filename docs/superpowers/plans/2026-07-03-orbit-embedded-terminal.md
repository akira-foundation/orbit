# Orbit Embedded Terminal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a real interactive PTY terminal per project, opened at the project root, exposed as a new tab in `ProjectDetail`, persisting in the background across tab switches until the app quits or the project is deleted.

**Architecture:** A new `internal/terminal` package holds a `Manager` (one PTY `Session` per project, mirroring the existing `runtime.Manager`/`services.Manager` shape). A reader goroutine per session drains the PTY into a 512KB ring buffer and emits chunks as Wails events. The frontend renders xterm.js in a new "Terminal" tab, replaying the buffer on mount and staying subscribed to live events while mounted.

**Tech Stack:** Go `github.com/creack/pty`, Wails v2 events (`EventsEmit`/`EventsOn`), React 19 + `@xterm/xterm` + `@xterm/addon-fit` (bun).

## Global Constraints

- Module path `orbit-app`; frontend package manager is **bun** (`bun add`, never npm/pnpm/yarn).
- No narrative comments; strict comment policy (see `~/.claude/skills/commit-guard`).
- No chained `else if` — use guard clauses / early returns.
- Every functional change ships tests; Go tests use real processes, no mocks.
- Commit protocol: `gofmt -w <changed files>`, run
  `bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh` and
  `scan-chained-if.sh` (both must exit 0), `go vet ./... && go test ./...`
  (`-short` for the full suite, network-gated tests separately), then
  `touch "${TMPDIR:-/tmp}/commit-guard/ok"` in its **own** Bash call, then
  `git commit` in a **separate** Bash call (the guard hook blocks a chained
  marker-touch + commit in one command).
- Ring buffer cap: 512KB per session, oldest bytes evicted (spec).
- Session lifecycle: dies only on app `shutdown()` or `DeleteProject` — never on tab switch or project stop (spec).
- Shell: `$SHELL` env var, fallback `/bin/zsh` (spec).

---

## File Structure

**Create:**
- `internal/terminal/manager.go` — `Manager`, `Session`, lifecycle methods
- `internal/terminal/manager_test.go`
- `internal/terminal/ringbuffer.go` — byte-capped ring buffer
- `internal/terminal/ringbuffer_test.go`
- `frontend/src/components/ProjectTerminal.tsx` — xterm.js UI
- `frontend/src/hooks/useProjectTerminal.ts` — buffer/event wiring (extracted so `ProjectTerminal.tsx` stays under ~120 lines)

**Modify:**
- `app.go` — construct `terminal.Manager`, wire into `startup`/`shutdown`/`DeleteProject`, add 4 Wails methods
- `frontend/src/api.ts` — 4 new wrapper functions
- `frontend/src/pages/ProjectDetail.tsx` — add `"terminal"` to `DETAIL_TABS` and render `<ProjectTerminal>`
- `frontend/package.json` — add `@xterm/xterm`, `@xterm/addon-fit`

---

## Task 1: Ring buffer

**Files:**
- Create: `internal/terminal/ringbuffer.go`
- Test: `internal/terminal/ringbuffer_test.go`

**Interfaces:**
- Produces:
  ```go
  type ringBuffer struct { ... }
  func newRingBuffer(capBytes int) *ringBuffer
  func (r *ringBuffer) Write(p []byte)
  func (r *ringBuffer) Bytes() []byte
  ```

- [ ] **Step 1: Write the failing test**

Create `internal/terminal/ringbuffer_test.go`:

```go
package terminal

import (
	"bytes"
	"testing"
)

func TestRingBufferAppendsUnderCap(t *testing.T) {
	rb := newRingBuffer(1024)
	rb.Write([]byte("hello "))
	rb.Write([]byte("world"))
	if got := rb.Bytes(); !bytes.Equal(got, []byte("hello world")) {
		t.Fatalf("bytes = %q", got)
	}
}

func TestRingBufferEvictsOldestOverCap(t *testing.T) {
	rb := newRingBuffer(10)
	rb.Write([]byte("0123456789"))
	rb.Write([]byte("abcde"))
	got := rb.Bytes()
	if len(got) != 10 {
		t.Fatalf("len = %d want 10", len(got))
	}
	if !bytes.Equal(got, []byte("56789abcde")) {
		t.Fatalf("bytes = %q want %q", got, "56789abcde")
	}
}

func TestRingBufferSingleWriteLargerThanCap(t *testing.T) {
	rb := newRingBuffer(4)
	rb.Write([]byte("abcdefgh"))
	if got := rb.Bytes(); !bytes.Equal(got, []byte("efgh")) {
		t.Fatalf("bytes = %q want %q", got, "efgh")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/terminal/... -run TestRingBuffer -v`
Expected: FAIL — package `terminal` / `newRingBuffer` undefined (directory doesn't exist yet).

- [ ] **Step 3: Write the implementation**

Create `internal/terminal/ringbuffer.go`:

```go
package terminal

import "sync"

type ringBuffer struct {
	mu   sync.Mutex
	cap  int
	data []byte
}

func newRingBuffer(capBytes int) *ringBuffer {
	return &ringBuffer{cap: capBytes, data: make([]byte, 0, capBytes)}
}

func (r *ringBuffer) Write(p []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(p) >= r.cap {
		r.data = append(r.data[:0], p[len(p)-r.cap:]...)
		return
	}

	r.data = append(r.data, p...)
	if over := len(r.data) - r.cap; over > 0 {
		r.data = append(r.data[:0], r.data[over:]...)
	}
}

func (r *ringBuffer) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]byte, len(r.data))
	copy(out, r.data)
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/terminal/... -run TestRingBuffer -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
gofmt -w internal/terminal/ringbuffer.go internal/terminal/ringbuffer_test.go
```
```bash
cd /Users/kid/Akira/Foundation/orbit && bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh && bash ~/.claude/skills/commit-guard/scripts/scan-chained-if.sh
```
```bash
touch "${TMPDIR:-/tmp}/commit-guard/ok"
```
```bash
cd /Users/kid/Akira/Foundation/orbit && git add internal/terminal/ringbuffer.go internal/terminal/ringbuffer_test.go && git commit -m "feat(terminal): byte-capped ring buffer for PTY scrollback"
```

---

## Task 2: PTY Manager

**Files:**
- Create: `internal/terminal/manager.go`
- Test: `internal/terminal/manager_test.go`

**Interfaces:**
- Consumes: `newRingBuffer(capBytes int) *ringBuffer`, `(*ringBuffer).Write([]byte)`, `(*ringBuffer).Bytes() []byte` (Task 1).
- Produces:
  ```go
  type Emitter interface { Emit(name string, data ...any) }
  type Manager struct { ... }
  func NewManager() *Manager
  func (m *Manager) SetEmitter(e Emitter)
  func (m *Manager) Start(projectID, path string) error
  func (m *Manager) Write(projectID string, data []byte) error
  func (m *Manager) Resize(projectID string, cols, rows int) error
  func (m *Manager) Buffer(projectID string) []byte
  func (m *Manager) Stop(projectID string)
  func (m *Manager) StopAll()
  func DataEventName(projectID string) string
  ```
  `Manager` accepts an `Emitter` (same shape as `runtime.Emitter`) so tests
  can inject a fake instead of touching Wails; production wiring in Task 4
  passes `runtime.NewWailsEmitter(ctx)` (already implements this interface —
  `Emit(name string, data ...any)`).

- [ ] **Step 1: Add the `creack/pty` dependency**

Run: `go get github.com/creack/pty@latest`

- [ ] **Step 2: Write the failing test**

Create `internal/terminal/manager_test.go`:

```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/terminal/... -run TestManager -v`
Expected: FAIL — `NewManager`, `fakeEmitter`, etc. undefined.

- [ ] **Step 4: Write the implementation**

Create `internal/terminal/manager.go`:

```go
package terminal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
)

const ringBufferCap = 512 * 1024

type Emitter interface {
	Emit(name string, data ...any)
}

type nopEmitter struct{}

func (nopEmitter) Emit(string, ...any) {}

func DataEventName(projectID string) string {
	return "terminal:data:" + projectID
}

type Session struct {
	cmd    *exec.Cmd
	pty    *os.File
	buffer *ringBuffer
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	emitter  Emitter
}

func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		emitter:  nopEmitter{},
	}
}

func (m *Manager) SetEmitter(e Emitter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if e == nil {
		e = nopEmitter{}
	}
	m.emitter = e
}

func shellPath() string {
	if s := os.Getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/zsh"
}

func (m *Manager) Start(projectID, path string) error {
	m.mu.Lock()
	if _, exists := m.sessions[projectID]; exists {
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	cmd := exec.Command(shellPath())
	cmd.Dir = path
	cmd.Env = os.Environ()

	f, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("terminal: start pty: %w", err)
	}

	sess := &Session{cmd: cmd, pty: f, buffer: newRingBuffer(ringBufferCap)}

	m.mu.Lock()
	m.sessions[projectID] = sess
	m.mu.Unlock()

	go m.readLoop(projectID, sess)
	return nil
}

func (m *Manager) readLoop(projectID string, sess *Session) {
	buf := make([]byte, 4096)
	for {
		n, err := sess.pty.Read(buf)
		if n > 0 {
			chunk := append([]byte(nil), buf[:n]...)
			sess.buffer.Write(chunk)
			m.mu.Lock()
			e := m.emitter
			m.mu.Unlock()
			e.Emit(DataEventName(projectID), string(chunk))
		}
		if err != nil {
			if err != io.EOF {
				return
			}
			return
		}
	}
}

func (m *Manager) Write(projectID string, data []byte) error {
	m.mu.Lock()
	sess, ok := m.sessions[projectID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("terminal: no session for %s", projectID)
	}
	_, err := sess.pty.Write(data)
	return err
}

func (m *Manager) Resize(projectID string, cols, rows int) error {
	m.mu.Lock()
	sess, ok := m.sessions[projectID]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("terminal: no session for %s", projectID)
	}
	return pty.Setsize(sess.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
}

func (m *Manager) Buffer(projectID string) []byte {
	m.mu.Lock()
	sess, ok := m.sessions[projectID]
	m.mu.Unlock()
	if !ok {
		return nil
	}
	return sess.buffer.Bytes()
}

func (m *Manager) Stop(projectID string) {
	m.mu.Lock()
	sess, ok := m.sessions[projectID]
	if ok {
		delete(m.sessions, projectID)
	}
	m.mu.Unlock()
	if !ok {
		return
	}
	_ = sess.cmd.Process.Kill()
	_ = sess.pty.Close()
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		m.Stop(id)
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/terminal/... -v -timeout 30s`
Expected: all PASS.

- [ ] **Step 6: Commit**

```bash
gofmt -w internal/terminal/manager.go internal/terminal/manager_test.go
```
```bash
cd /Users/kid/Akira/Foundation/orbit && go mod tidy && bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh && bash ~/.claude/skills/commit-guard/scripts/scan-chained-if.sh
```
```bash
touch "${TMPDIR:-/tmp}/commit-guard/ok"
```
```bash
cd /Users/kid/Akira/Foundation/orbit && git add internal/terminal/manager.go internal/terminal/manager_test.go go.mod go.sum && git commit -m "feat(terminal): PTY session manager with ring-buffer scrollback"
```

---

## Task 3: Wails API + app wiring

**Files:**
- Modify: `app.go`

**Interfaces:**
- Consumes: `terminal.NewManager()`, `(*terminal.Manager).SetEmitter(Emitter)`, `Start(projectID, path string) error`, `Write(projectID string, data []byte) error`, `Resize(projectID string, cols, rows int) error`, `Buffer(projectID string) []byte`, `Stop(projectID string)`, `StopAll()` (Task 2). `runtime.NewWailsEmitter(ctx)` (existing) satisfies `terminal.Emitter`.
- Produces (Wails-bound methods):
  ```go
  func (a *App) TerminalStart(projectID string) error
  func (a *App) TerminalWrite(projectID, data string) error
  func (a *App) TerminalResize(projectID string, cols, rows int) error
  func (a *App) TerminalBuffer(projectID string) (string, error)
  ```

- [ ] **Step 1: Add the field and import**

In `app.go`, add to imports:

```go
	"orbit-app/internal/terminal"
```

Add to the `App` struct (alongside `services *services.Manager`):

```go
	terminals *terminal.Manager
```

- [ ] **Step 2: Construct and wire in `startup()`**

Locate where `a.runtime.SetEmitter(runtime.NewWailsEmitter(ctx))` is called
in `startup()`. Immediately after it, add:

```go
	a.terminals = terminal.NewManager()
	a.terminals.SetEmitter(runtime.NewWailsEmitter(ctx))
```

- [ ] **Step 3: Wire into `shutdown()`**

In `shutdown()`, before `a.runtime.Close()`, add:

```go
	if a.terminals != nil {
		a.terminals.StopAll()
	}
```

- [ ] **Step 4: Wire into `DeleteProject`**

Find `func (a *App) DeleteProject(id string) error`. Before the existing
`return a.service.Delete(a.ctx, id)` line, add:

```go
	a.terminals.Stop(id)
```

- [ ] **Step 5: Add the four Wails methods**

Append near the other project-scoped methods in `app.go` (e.g. after
`ServiceSetup`):

```go
func (a *App) TerminalStart(projectID string) error {
	proj, err := a.service.Get(a.ctx, projectID)
	if err != nil {
		return err
	}
	return a.terminals.Start(projectID, proj.Path)
}

func (a *App) TerminalWrite(projectID, data string) error {
	return a.terminals.Write(projectID, []byte(data))
}

func (a *App) TerminalResize(projectID string, cols, rows int) error {
	return a.terminals.Resize(projectID, cols, rows)
}

func (a *App) TerminalBuffer(projectID string) (string, error) {
	return string(a.terminals.Buffer(projectID)), nil
}
```

- [ ] **Step 6: Build and regenerate bindings**

Run: `go build ./... && go vet ./...`
Expected: clean.

Run: `wails generate module`
Verify: `grep -o "TerminalStart" frontend/wailsjs/go/main/App.d.ts | head -1` prints `TerminalStart`.

- [ ] **Step 7: Run full Go suite**

Run: `go test ./... -short`
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
gofmt -w app.go
```
```bash
cd /Users/kid/Akira/Foundation/orbit && bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh && bash ~/.claude/skills/commit-guard/scripts/scan-chained-if.sh
```
```bash
touch "${TMPDIR:-/tmp}/commit-guard/ok"
```
```bash
cd /Users/kid/Akira/Foundation/orbit && git add app.go frontend/wailsjs && git commit -m "feat(app): wire terminal manager into lifecycle + expose Wails API"
```

---

## Task 4: Frontend — xterm.js dependency + api.ts wrappers

**Files:**
- Modify: `frontend/package.json`, `frontend/src/api.ts`

**Interfaces:**
- Consumes: Wails bindings `TerminalStart`, `TerminalWrite`, `TerminalResize`, `TerminalBuffer` (Task 3).
- Produces:
  ```ts
  export const api = {
    // ...existing
    terminalStart: (projectId: string) => Promise<void>
    terminalWrite: (projectId: string, data: string) => Promise<void>
    terminalResize: (projectId: string, cols: number, rows: number) => Promise<void>
    terminalBuffer: (projectId: string) => Promise<string>
  }
  ```

- [ ] **Step 1: Add the frontend dependencies**

Run: `cd frontend && bun add @xterm/xterm @xterm/addon-fit`

- [ ] **Step 2: Add wrappers to `api.ts`**

In `frontend/src/api.ts`, add to the import block from
`"../wailsjs/go/main/App"`:

```ts
  TerminalStart,
  TerminalWrite,
  TerminalResize,
  TerminalBuffer,
```

Add to the exported `api` object (after `clearServicesData`):

```ts
  terminalStart: (projectId: string): Promise<void> =>
    TerminalStart(projectId),
  terminalWrite: (projectId: string, data: string): Promise<void> =>
    TerminalWrite(projectId, data),
  terminalResize: (
    projectId: string,
    cols: number,
    rows: number,
  ): Promise<void> => TerminalResize(projectId, cols, rows),
  terminalBuffer: (projectId: string): Promise<string> =>
    TerminalBuffer(projectId),
```

- [ ] **Step 3: Build**

Run: `cd frontend && bun run build`
Expected: no TS errors (bindings for the 4 methods exist from Task 3 Step 6).

- [ ] **Step 4: Commit**

```bash
cd /Users/kid/Akira/Foundation/orbit && bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh && bash ~/.claude/skills/commit-guard/scripts/scan-chained-if.sh
```
```bash
touch "${TMPDIR:-/tmp}/commit-guard/ok"
```
```bash
cd /Users/kid/Akira/Foundation/orbit && git add frontend/package.json frontend/bun.lock frontend/src/api.ts && git commit -m "feat(ui): add xterm.js dependency and terminal api wrappers"
```

---

## Task 5: Frontend — `useProjectTerminal` hook + `ProjectTerminal` component

**Files:**
- Create: `frontend/src/hooks/useProjectTerminal.ts`
- Create: `frontend/src/components/ProjectTerminal.tsx`

**Interfaces:**
- Consumes: `api.terminalStart/terminalWrite/terminalResize/terminalBuffer`
  (Task 4), `useWailsEvent` (existing, `frontend/src/hooks/useWailsEvent.ts`),
  `@xterm/xterm` `Terminal`, `@xterm/addon-fit` `FitAddon` (Task 4 deps).
- Produces:
  ```ts
  // useProjectTerminal.ts
  export function useProjectTerminal(
    projectId: string,
    term: import("@xterm/xterm").Terminal | null,
  ): void
  // ProjectTerminal.tsx
  export function ProjectTerminal({ projectId }: { projectId: string }): JSX.Element
  ```

The hook owns the event-queue-before-buffer-replay race handling described
in the design: subscribe to the live data event immediately, queue
incoming chunks in a ref until the buffer fetch resolves, write the buffer
then flush the queue, then switch to writing new events directly.

- [ ] **Step 1: Write `useProjectTerminal.ts`**

Create `frontend/src/hooks/useProjectTerminal.ts`:

```ts
import { useEffect, useRef } from "react";
import type { Terminal } from "@xterm/xterm";
import { api } from "../api";
import { useWailsEvent } from "./useWailsEvent";

export function useProjectTerminal(projectId: string, term: Terminal | null) {
  const replayedRef = useRef(false);
  const queueRef = useRef<string[]>([]);
  const termRef = useRef<Terminal | null>(null);
  termRef.current = term;

  useWailsEvent<string>(`terminal:data:${projectId}`, (chunk) => {
    if (!replayedRef.current) {
      queueRef.current.push(chunk);
      return;
    }
    termRef.current?.write(chunk);
  });

  useEffect(() => {
    if (!term) return;
    replayedRef.current = false;
    queueRef.current = [];

    let cancelled = false;

    async function boot() {
      await api.terminalStart(projectId);
      const buffer = await api.terminalBuffer(projectId);
      if (cancelled) return;
      if (buffer) term!.write(buffer);
      for (const chunk of queueRef.current) {
        term!.write(chunk);
      }
      queueRef.current = [];
      replayedRef.current = true;
    }

    boot();

    const onData = term.onData((data) => {
      api.terminalWrite(projectId, data);
    });

    return () => {
      cancelled = true;
      onData.dispose();
    };
  }, [projectId, term]);
}
```

- [ ] **Step 2: Write `ProjectTerminal.tsx`**

Create `frontend/src/components/ProjectTerminal.tsx`:

```tsx
import { useEffect, useRef, useState } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import "@xterm/xterm/css/xterm.css";
import { api } from "../api";
import { useProjectTerminal } from "../hooks/useProjectTerminal";

export function ProjectTerminal({ projectId }: { projectId: string }) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const [term, setTerm] = useState<Terminal | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const instance = new Terminal({
      convertEol: true,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
      fontSize: 12.5,
      theme: { background: "#0c0d12" },
    });
    const fit = new FitAddon();
    instance.loadAddon(fit);
    instance.open(containerRef.current);
    fit.fit();
    fitRef.current = fit;
    setTerm(instance);

    return () => {
      instance.dispose();
      setTerm(null);
    };
  }, []);

  useProjectTerminal(projectId, term);

  useEffect(() => {
    if (!term || !containerRef.current) return;
    const observer = new ResizeObserver(() => {
      fitRef.current?.fit();
      api.terminalResize(projectId, term.cols, term.rows);
    });
    observer.observe(containerRef.current);
    return () => observer.disconnect();
  }, [term, projectId]);

  return (
    <div
      ref={containerRef}
      className="h-[420px] rounded-lg overflow-hidden border border-white/10 bg-[#0c0d12] p-2"
    />
  );
}
```

- [ ] **Step 3: Build**

Run: `cd frontend && bun run build`
Expected: no TS errors.

- [ ] **Step 4: Commit**

```bash
cd /Users/kid/Akira/Foundation/orbit && bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh && bash ~/.claude/skills/commit-guard/scripts/scan-chained-if.sh
```
```bash
touch "${TMPDIR:-/tmp}/commit-guard/ok"
```
```bash
cd /Users/kid/Akira/Foundation/orbit && git add frontend/src/hooks/useProjectTerminal.ts frontend/src/components/ProjectTerminal.tsx && git commit -m "feat(ui): xterm.js terminal component with buffer replay"
```

---

## Task 6: Wire the Terminal tab into ProjectDetail

**Files:**
- Modify: `frontend/src/pages/ProjectDetail.tsx`

**Interfaces:**
- Consumes: `ProjectTerminal` (Task 5), existing `DETAIL_TABS` array and
  tab-rendering block established in the prior "tab project detail
  sections" change (commit `c1da2c4`).

- [ ] **Step 1: Import the component**

In `frontend/src/pages/ProjectDetail.tsx`, add:

```ts
import { ProjectTerminal } from "../components/ProjectTerminal";
```

- [ ] **Step 2: Extend the tab type and `DETAIL_TABS` array**

Change:

```ts
const DETAIL_TABS: {
  id: "details" | "services" | "logs";
```

to:

```ts
const DETAIL_TABS: {
  id: "details" | "services" | "logs" | "terminal";
```

And add an entry to the array (after the `logs` entry, reusing the already
imported `Terminal` icon from `lucide-react` used elsewhere in this file):

```ts
  { id: "terminal", label: "Terminal", subtitle: "Interactive shell at the project root", icon: Terminal },
```

Update the local tab state type to match:

```ts
const [tab, setTab] = useState<"details" | "services" | "logs" | "terminal">("details");
```

- [ ] **Step 3: Render the panel**

Inside the `<Card>` block where `tab === "logs"` is rendered, add a sibling
branch:

```tsx
{tab === "terminal" && <ProjectTerminal projectId={id} />}
```

- [ ] **Step 4: Build**

Run: `cd frontend && bun run build`
Expected: no TS errors.

- [ ] **Step 5: Commit**

```bash
cd /Users/kid/Akira/Foundation/orbit && bash ~/.claude/skills/commit-guard/scripts/scan-comments.sh && bash ~/.claude/skills/commit-guard/scripts/scan-chained-if.sh
```
```bash
touch "${TMPDIR:-/tmp}/commit-guard/ok"
```
```bash
cd /Users/kid/Akira/Foundation/orbit && git add frontend/src/pages/ProjectDetail.tsx && git commit -m "feat(ui): add Terminal tab to project detail"
```

---

## Task 7: Manual end-to-end verification

- [ ] **Step 1:** `wails dev` (or reuse the already-running dev instance).
- [ ] **Step 2:** Open a project, click the new "Terminal" tab. A shell prompt appears, `pwd` prints the project's path.
- [ ] **Step 3:** Run a full-screen program, e.g. `htop` or `vim`; confirm it renders and responds to keys/resizing the window.
- [ ] **Step 4:** Switch to "Details" tab, back to "Terminal" — the screen is exactly as left (scrollback replay).
- [ ] **Step 5:** Switch to a different project, back — confirm this project's own terminal (not shared) and its own scrollback.
- [ ] **Step 6:** Stop the project's dev server (Start/Stop button in the hero) — terminal session stays alive, still usable.
- [ ] **Step 7:** Delete the project — confirm no orphaned shell process remains (`ps aux | grep <shell>` before/after).
- [ ] **Step 8:** Quit the Orbit app — confirm all PTY child processes are gone (`ps aux | grep zsh` or equivalent before/after quit).
- [ ] **Step 9:** `go vet ./... && go test ./... -short && (cd frontend && bun run build)` all green.

---

## Self-Review

**Spec coverage:** PTY model → Task 2; `$SHELL` fallback → Task 2 `shellPath`; Terminal tab location → Task 6; one session per project → `Manager.sessions` keyed by projectID (Task 2); lifecycle (app quit / project delete only) → Task 3 Steps 3–4; ring buffer replay + race handling → Task 5; cwd = project.Path → Task 3 `TerminalStart` reading `proj.Path`; xterm.js + addon-fit dependency → Task 4; testing without mocks → Task 2 uses real PTYs/processes throughout.

**Placeholders:** none — every code step has complete, runnable content.

**Type consistency:** `Manager.Start(projectID, path string) error` (Task 2) matches the call in `app.go`'s `TerminalStart` (Task 3). `DataEventName(projectID string) string` (Task 2) matches the event name string built ad hoc in `useProjectTerminal.ts` (`` `terminal:data:${projectId}` ``) — both produce `terminal:data:<id>`, kept as a literal template on the frontend since there's no shared codegen between Go consts and TS, consistent with how `runtime`'s event names (`runtime:starting` etc.) are already hardcoded as string literals in the existing frontend code. `terminal.Emitter` interface shape (`Emit(name string, data ...any)`) matches `runtime.Emitter` exactly, so `runtime.NewWailsEmitter(ctx)` satisfies both without adapter code.
