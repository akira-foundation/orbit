package bindings

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"orbit-app/internal/config"
	"orbit-app/internal/projects"
	"orbit-app/internal/runtime"
	"orbit-app/internal/services"
	"orbit-app/internal/terminal"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type AnalyzeResult struct {
	Name            string            `json:"name"`
	Path            string            `json:"path"`
	PackageManager  string            `json:"packageManager"`
	Framework       string            `json:"framework"`
	DevCommand      string            `json:"devCommand"`
	DevPort         int               `json:"devPort"`
	Scripts         map[string]string `json:"scripts"`
	SuggestedSlug   string            `json:"suggestedSlug"`
	SuggestedDomain string            `json:"suggestedDomain"`
}

type ProjectURL struct {
	URL    string `json:"url"`
	Domain string `json:"domain"`
	Port   string `json:"port"`
}

type Projects struct {
	ctx       context.Context
	cfg       *config.Config
	service   *projects.Service
	runtime   runtime.Manager
	svcStore  *services.Store
	svcConfig *services.ConfigStore
	terminals *terminal.Manager
}

func NewProjects() *Projects { return &Projects{} }

func (p *Projects) Attach(d Deps) {
	p.ctx = d.Ctx
	p.cfg = d.Cfg
	p.service = d.Service
	p.runtime = d.Runtime
	p.svcStore = d.SvcStore
	p.svcConfig = d.SvcConfig
	p.terminals = d.Terminals
}

func (p *Projects) PathNeedsInstall(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(path, "node_modules"))
	return err != nil || !st.IsDir()
}

func (p *Projects) InstallProject(id string) error {
	return p.runtime.Install(p.ctx, id)
}

func (p *Projects) AnalyzePath(path string) (*AnalyzeResult, error) {
	res, err := p.service.AnalyzePath(path)
	if err != nil {
		return nil, err
	}
	slug := projects.Slugify(res.Name)
	return &AnalyzeResult{
		Name:            res.Name,
		Path:            res.Path,
		PackageManager:  res.PackageManager,
		Framework:       res.Framework,
		DevCommand:      res.DevCommand,
		DevPort:         res.DevPort,
		Scripts:         res.Scripts,
		SuggestedSlug:   slug,
		SuggestedDomain: p.cfg.DomainFor(slug),
	}, nil
}

func (p *Projects) AddProject(path string) (*projects.Project, error) {
	proj, err := p.service.Create(p.ctx, path)
	if err != nil {
		p.runtime.RecordEvent("", runtime.LevelError, runtime.SourceProject,
			fmt.Sprintf("add project %q failed: %v", path, err))
		return nil, err
	}
	p.runtime.RecordEvent(proj.ID, runtime.LevelInfo, runtime.SourceProject,
		fmt.Sprintf("project added: %s (%s)", proj.Name, proj.LocalDomain))
	for _, engine := range p.svcConfig.DefaultEngines() {
		_ = p.svcStore.SetEnabled(p.ctx, proj.ID, engine, true)
	}
	return proj, nil
}

func (p *Projects) ListProjects() ([]projects.Project, error) {
	return p.service.List(p.ctx)
}

func (p *Projects) GetProject(id string) (*projects.Project, error) {
	return p.service.Get(p.ctx, id)
}

func (p *Projects) SetProjectSecure(id string, secure bool) error {
	return p.service.SetSecure(p.ctx, id, secure)
}

func (p *Projects) DeleteProject(id string) error {
	p.runtime.RecordEvent(id, runtime.LevelWarn, runtime.SourceProject, "project deleted")
	p.terminals.Stop(id)
	return p.service.Delete(p.ctx, id)
}

func (p *Projects) StartProject(id string) error {
	return p.runtime.Start(p.ctx, id)
}

func (p *Projects) StopProject(id string) error {
	return p.runtime.Stop(p.ctx, id)
}

func (p *Projects) RestartProject(id string) error {
	return p.runtime.Restart(p.ctx, id)
}

func (p *Projects) SelectProjectFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(p.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select project folder",
	})
}

func (p *Projects) ProjectURL(id string) (*ProjectURL, error) {
	proj, err := p.service.Get(p.ctx, id)
	if err != nil {
		return nil, err
	}
	return &ProjectURL{
		URL:    p.cfg.URLFor(proj.LocalDomain),
		Domain: proj.LocalDomain,
		Port:   p.cfg.ProxyPort(),
	}, nil
}

func (p *Projects) OpenProject(id string) error {
	u, err := p.ProjectURL(id)
	if err != nil {
		return err
	}
	wailsruntime.BrowserOpenURL(p.ctx, u.URL)
	return nil
}
