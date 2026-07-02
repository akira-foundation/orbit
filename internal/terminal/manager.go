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
