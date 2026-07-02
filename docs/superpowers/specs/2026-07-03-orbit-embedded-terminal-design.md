# Orbit Embedded Terminal Design

Date: 2026-07-03
Status: Approved

## Goal

Add a real, interactive terminal to each project's detail view, opened at
the project's root path. Supports full-screen apps (vim, htop), colors,
Ctrl+C, and persists in the background across tab switches so the user can
leave and come back to a running command.

## Non-goals

- Multiple terminal tabs per project (one PTY session per project for now).
- Remote/SSH terminals — local shell only.
- Command history search, tmux-style panes, or session naming.
- Access control beyond what the user already has on their own machine —
  this exposes an interactive shell in the project directory, which is
  exactly the point (equivalent to opening Terminal.app there). No
  additional sandboxing; the user already has full filesystem access.

## Decisions

| Topic | Decision |
|-------|----------|
| Terminal model | Real PTY (not a single-command executor) — supports interactive/full-screen programs. |
| Shell | The user's `$SHELL` (falls back to `/bin/zsh` if unset), matching their native terminal experience (aliases, prompt customization). |
| Location in UI | New "Terminal" tab in `ProjectDetail`, alongside Details/Services/Logs. |
| Session count | One PTY session per project. |
| Session lifecycle | Persists across tab switches and project stop/start. Dies only when Orbit quits (`shutdown()` kills all sessions) or the project is deleted (`DeleteProject` kills that project's session). |
| Reconnect behavior | Full scrollback replay — an in-memory ring buffer (512KB cap per session, oldest bytes evicted) is fetched and replayed when the tab is reopened, so the screen looks exactly as it did before navigating away. |
| Working directory | `project.Path`, set as the PTY command's `cwd`. |

## Architecture

### Backend — `internal/terminal/`

Mirrors the existing `runtime.Manager` / `services.Manager` pattern: a
`Manager` holding `map[string]*Session` behind a mutex.

- `Session` wraps a PTY (via `github.com/creack/pty`, the standard Go PTY
  library) running the user's shell with `cmd.Dir = project.Path`. A reader
  goroutine continuously drains the PTY into:
  1. A ring buffer (512KB, byte-capped, oldest evicted) — the scrollback.
  2. A Wails event `terminal:data:<projectID>` carrying the raw chunk, for
     any currently-open terminal tab to render live.
- `Manager.Start(ctx, projectID, path string) error` — idempotent; a second
  call while a session is alive is a no-op.
- `Manager.Write(projectID string, data []byte) error` — forwards keystrokes
  to the PTY's stdin.
- `Manager.Resize(projectID string, cols, rows int) error` — calls
  `pty.Setsize`.
- `Manager.Buffer(projectID string) []byte` — returns the current ring
  buffer contents for replay.
- `Manager.Stop(projectID string)` — kills one session (used on project
  delete).
- `Manager.StopAll()` — kills every session (used on app shutdown).

### Wails API (`app.go`)

```go
func (a *App) TerminalStart(projectID string) error
func (a *App) TerminalWrite(projectID, data string) error
func (a *App) TerminalResize(projectID string, cols, rows int) error
func (a *App) TerminalBuffer(projectID string) (string, error)
```

Wired into `startup()` (construct `a.terminals = terminal.NewManager()`),
`shutdown()` (`a.terminals.StopAll()`), and `DeleteProject`
(`a.terminals.Stop(id)` alongside existing cleanup).

### Frontend

New dependency: `@xterm/xterm` + `@xterm/addon-fit` (the standard browser
terminal emulator — same one VS Code's web terminal and Hyper use).

`frontend/src/components/ProjectTerminal.tsx`:
- On mount: subscribe to the `terminal:data:<projectId>` Wails event
  *before* fetching the buffer (avoids a race where output emitted between
  fetch and subscribe would be lost) — events arriving during the fetch are
  queued and flushed immediately after the buffer replay writes to the
  xterm instance.
- Calls `api.terminalStart(projectId)`, then `api.terminalBuffer(projectId)`
  for the replay.
- Keystrokes (xterm `onData`) forward to `api.terminalWrite`.
- A `ResizeObserver` + the fit addon recompute cols/rows on container
  resize and call `api.terminalResize`.
- Unmounting the component only tears down the xterm.js UI instance — the
  server-side PTY keeps running untouched.

New tab entry in `frontend/src/pages/ProjectDetail.tsx`'s `DETAIL_TABS`
array: `{ id: "terminal", label: "Terminal", subtitle: "Interactive shell at the project root", icon: TerminalIcon }`.

## Testing

Go, no mocks:
- `Manager.Start` spawns a real PTY, `Write` + reading the ring buffer
  round-trips a command's output (e.g. write `echo hi\n`, assert buffer
  contains `hi`).
- `Resize` doesn't error against a live session.
- `Stop`/`StopAll` actually terminate the underlying process (check exit).
- Idempotent `Start` (second call returns nil, doesn't spawn a second PTY —
  assert same PID).

Frontend: manual verification (PTY interaction isn't practical to unit test
against real xterm.js rendering) — covered in the implementation plan's
manual verification task.
