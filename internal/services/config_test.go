package services

import (
	"context"
	"testing"
)

func TestConfigStorePersistsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	s := LoadConfig(dir)
	if !s.Get().AutoManage {
		t.Fatal("expected AutoManage default true")
	}

	if err := s.Save(Config{
		AutoManage:      false,
		IdleStopMinutes: 15,
		Defaults:        map[string]bool{"mailpit": true},
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	reloaded := LoadConfig(dir).Get()
	if reloaded.AutoManage || reloaded.IdleStopMinutes != 15 || !reloaded.Defaults["mailpit"] {
		t.Fatalf("reloaded = %+v", reloaded)
	}
	if got := LoadConfig(dir).DefaultEngines(); len(got) != 1 || got[0] != "mailpit" {
		t.Fatalf("default engines = %v", got)
	}
}

func TestOnProjectStartSkipsWhenAutoManageOff(t *testing.T) {
	m, _ := newManagerForTest(t, freePort(t))
	_ = m.cfg.Save(Config{AutoManage: false})
	_ = m.acq.store.SetEnabled(context.Background(), "p1", "mailpit", true)

	if err := m.OnProjectStart(context.Background(), "p1", t.TempDir()); err != nil {
		t.Fatalf("on start: %v", err)
	}
	if got := m.Status("mailpit"); got.Status != "stopped" {
		t.Fatalf("expected service not auto-started, got %+v", got)
	}
}
