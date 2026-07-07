package runtime

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const (
	EvtStarting = "runtime:starting"
	EvtRunning  = "runtime:running"
	EvtStopped  = "runtime:stopped"
	EvtError    = "runtime:error"
	EvtLog      = "runtime:log"
	EvtSnapshot = "runtime:snapshot"
	EvtCrash    = "runtime:crash"
)

type Emitter interface {
	Emit(name string, data ...any)
}

type WailsEmitter struct {
	ctx context.Context
}

func NewWailsEmitter(ctx context.Context) *WailsEmitter {
	return &WailsEmitter{ctx: ctx}
}

func (e *WailsEmitter) Emit(name string, data ...any) {
	if e.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(e.ctx, name, data...)
}

type nopEmitter struct{}

func (nopEmitter) Emit(string, ...any) {}

type LogEvent struct {
	ProjectID string  `json:"projectId"`
	Line      LogLine `json:"line"`
}

type StatusEvent struct {
	ProjectID string   `json:"projectId"`
	Snapshot  Snapshot `json:"snapshot"`
}
