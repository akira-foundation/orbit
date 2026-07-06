package bindings

import (
	"context"

	"orbit-app/internal/projects"
	"orbit-app/internal/terminal"
)

type Terminal struct {
	ctx       context.Context
	terminals *terminal.Manager
	service   *projects.Service
}

func NewTerminal() *Terminal { return &Terminal{} }

func (t *Terminal) Attach(d Deps) {
	t.ctx = d.Ctx
	t.terminals = d.Terminals
	t.service = d.Service
}

func (t *Terminal) TerminalStart(projectID string) error {
	proj, err := t.service.Get(t.ctx, projectID)
	if err != nil {
		return err
	}
	return t.terminals.Start(projectID, proj.Path)
}

func (t *Terminal) TerminalWrite(projectID, data string) error {
	return t.terminals.Write(projectID, []byte(data))
}

func (t *Terminal) TerminalResize(projectID string, cols, rows int) error {
	return t.terminals.Resize(projectID, cols, rows)
}

func (t *Terminal) TerminalBuffer(projectID string) (string, error) {
	return string(t.terminals.Buffer(projectID)), nil
}
