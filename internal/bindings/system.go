package bindings

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"orbit-app/internal/config"
	"orbit-app/internal/git"
	"orbit-app/internal/projects"
	"orbit-app/internal/system"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type System struct {
	ctx     context.Context
	cfg     *config.Config
	service *projects.Service
}

func NewSystem() *System { return &System{} }

func (s *System) Attach(d Deps) {
	s.ctx = d.Ctx
	s.cfg = d.Cfg
	s.service = d.Service
}

func (s *System) SystemConfig() *config.Config {
	return s.cfg
}

func (s *System) SystemSaveConfig(cfg *config.Config) error {
	return s.cfg.Save()
}

func (s *System) OpenURL(url string) error {
	if url == "" {
		return fmt.Errorf("empty url")
	}
	wailsruntime.BrowserOpenURL(s.ctx, url)
	return nil
}

func (s *System) GitStatus(id string) (*git.Info, error) {
	p, err := s.service.Get(s.ctx, id)
	if err != nil {
		return nil, err
	}
	info := git.Status(s.ctx, p.Path)
	return &info, nil
}

func (s *System) RevealInFinder(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	return exec.Command("open", path).Start()
}

func (s *System) SystemStatus() system.Status {
	return system.Check()
}

func (s *System) SystemUninstall() error {
	return system.Uninstall()
}

func (s *System) TrustCA() error {
	return system.TrustCA()
}

func (s *System) UntrustCA() error {
	return system.UntrustCA()
}

func (s *System) SetLaunchAtLogin(enabled bool) error {
	return system.SetLaunchAtLogin(enabled)
}

func (s *System) SystemSetup() error {
	cwd, _ := os.Getwd()
	if err := system.Install(cwd); err != nil {
		exe, eerr := os.Executable()
		if eerr == nil {
			return system.Install(exe)
		}
		return err
	}
	return nil
}
