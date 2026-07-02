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
