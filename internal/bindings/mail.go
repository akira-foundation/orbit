package bindings

import (
	"context"
	"fmt"
	"strings"
	"time"

	"orbit-app/internal/services"
	"orbit-app/internal/services/mailpit"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type Mail struct {
	ctx      context.Context
	services *services.Manager
}

func NewMail() *Mail { return &Mail{} }

func (m *Mail) Attach(d Deps) {
	m.ctx = d.Ctx
	m.services = d.Services
}

func (m *Mail) client() (*mailpit.Client, error) {
	base, ok := m.services.APIBase("mailpit")
	if !ok {
		return nil, fmt.Errorf("mailpit is not running")
	}
	return mailpit.NewClient(base), nil
}

func (m *Mail) MailList(start, limit int) (mailpit.ListResult, error) {
	c, err := m.client()
	if err != nil {
		return mailpit.ListResult{}, err
	}
	return c.List(m.ctx, start, limit)
}

func (m *Mail) MailGet(id string) (mailpit.Message, error) {
	c, err := m.client()
	if err != nil {
		return mailpit.Message{}, err
	}
	return c.Get(m.ctx, id)
}

func (m *Mail) MailSetRead(ids []string, read bool) error {
	c, err := m.client()
	if err != nil {
		return err
	}
	return c.SetRead(m.ctx, ids, read)
}

func (m *Mail) MailDelete(ids []string) error {
	c, err := m.client()
	if err != nil {
		return err
	}
	return c.Delete(m.ctx, ids)
}

func (m *Mail) MailDeleteAll() error {
	c, err := m.client()
	if err != nil {
		return err
	}
	return c.Delete(m.ctx, nil)
}

func (m *Mail) MailPart(id, partID string) (mailpit.PartData, error) {
	c, err := m.client()
	if err != nil {
		return mailpit.PartData{}, err
	}
	data, contentType, err := c.Part(m.ctx, id, partID)
	if err != nil {
		return mailpit.PartData{}, err
	}
	return mailpit.PartData{ContentType: contentType, Data: data}, nil
}

func (m *Mail) Watch(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		base, ok := m.services.APIBase("mailpit")
		if !ok {
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
			continue
		}
		wsURL := "ws" + strings.TrimPrefix(base, "http") + "/api/events"
		_ = mailpit.Watch(ctx, wsURL, func(ev mailpit.Event) {
			switch ev.Type {
			case "new":
				wailsruntime.EventsEmit(ctx, "mail:new", ev.Data)
			case "update":
				wailsruntime.EventsEmit(ctx, "mail:update", ev.Data)
			case "delete":
				wailsruntime.EventsEmit(ctx, "mail:delete", ev.Data)
			case "truncate":
				wailsruntime.EventsEmit(ctx, "mail:truncate")
			}
		})
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}
