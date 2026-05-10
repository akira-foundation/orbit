package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"orbit-app/internal/analyzer"
	"orbit-app/internal/config"
	"orbit-app/internal/database"
	"orbit-app/internal/projects"
	"orbit-app/internal/proxy"
	"orbit-app/internal/runtime"
	"orbit-app/internal/system"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	cfg         *config.Config
	db          *sql.DB
	service     *projects.Service
	runtime     runtime.Manager
	registry    proxy.Manager
	proxyServer *proxy.Server
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("config: %w", err))
	}
	a.cfg = cfg

	sqlDB, queries, err := database.Open(cfg.DBPath)
	if err != nil {
		panic(fmt.Errorf("database: %w", err))
	}
	a.db = sqlDB

	repo := projects.NewRepository(sqlDB, queries)
	a.service = projects.NewService(repo, analyzer.New(), cfg.DomainSuffix)
	a.runtime = runtime.New(a.service, sqlDB)
	a.runtime.SetEmitter(runtime.NewWailsEmitter(ctx))
	a.registry = proxy.New(a.service, cfg.DomainSuffix)

	router := proxy.NewRouter(a.registry, a.runtime)
	recovery := proxy.NewRecoveryHandler(a.registry, a.runtime)
	a.proxyServer = proxy.NewServer(cfg.ProxyAddr, router, recovery)
	go func() {
		if err := a.proxyServer.ListenAndServe(); err != nil {
			log.Printf("[proxy] server error: %v", err)
		}
	}()
}

func (a *App) shutdown(_ context.Context) {
	if a.proxyServer != nil {
		shCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = a.proxyServer.Shutdown(shCtx)
		cancel()
	}
	if a.runtime != nil {
		a.runtime.StopAll()
	}
	if a.db != nil {
		_ = a.db.Close()
	}
}

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

func (a *App) AnalyzePath(path string) (*AnalyzeResult, error) {
	res, err := a.service.AnalyzePath(path)
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
		SuggestedDomain: a.cfg.DomainFor(slug),
	}, nil
}

func (a *App) AddProject(path string) (*projects.Project, error) {
	p, err := a.service.Create(a.ctx, path)
	if err != nil {
		a.runtime.RecordEvent("", runtime.LevelError, runtime.SourceProject,
			fmt.Sprintf("add project %q failed: %v", path, err))
		return nil, err
	}
	a.runtime.RecordEvent(p.ID, runtime.LevelInfo, runtime.SourceProject,
		fmt.Sprintf("project added: %s (%s)", p.Name, p.LocalDomain))
	return p, nil
}

func (a *App) ListProjects() ([]projects.Project, error) {
	return a.service.List(a.ctx)
}

func (a *App) GetProject(id string) (*projects.Project, error) {
	return a.service.Get(a.ctx, id)
}

func (a *App) DeleteProject(id string) error {
	a.runtime.RecordEvent(id, runtime.LevelWarn, runtime.SourceProject,
		"project deleted")
	return a.service.Delete(a.ctx, id)
}

func (a *App) StartProject(id string) error {
	return a.runtime.Start(a.ctx, id)
}

func (a *App) StopProject(id string) error {
	return a.runtime.Stop(a.ctx, id)
}

func (a *App) RestartProject(id string) error {
	return a.runtime.Restart(a.ctx, id)
}

func (a *App) RuntimeStatus(id string) runtime.Snapshot {
	return a.runtime.Status(id)
}

func (a *App) RuntimeLogs(id string) []runtime.LogLine {
	return a.runtime.Logs(id)
}

func (a *App) RuntimeLogsHistory(id string, sinceTs int64, limit int) []runtime.LogLine {
	return a.runtime.LogsHistory(id, sinceTs, limit)
}

func (a *App) RuntimeMetrics(id string) []runtime.Sample {
	return a.runtime.Metrics(id)
}

func (a *App) RuntimeMetricsAll() map[string][]runtime.Sample {
	return a.runtime.MetricsAll()
}

func (a *App) SelectProjectFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select project folder",
	})
}

type ProjectURL struct {
	URL    string `json:"url"`
	Domain string `json:"domain"`
	Port   string `json:"port"`
}

func (a *App) ProjectURL(id string) (*ProjectURL, error) {
	p, err := a.service.Get(a.ctx, id)
	if err != nil {
		return nil, err
	}
	return &ProjectURL{
		URL:    a.cfg.URLFor(p.LocalDomain),
		Domain: p.LocalDomain,
		Port:   a.cfg.ProxyPort(),
	}, nil
}

func (a *App) OpenProject(id string) error {
	u, err := a.ProjectURL(id)
	if err != nil {
		return err
	}
	wailsruntime.BrowserOpenURL(a.ctx, u.URL)
	return nil
}

func (a *App) SystemStatus() system.Status {
	return system.Check()
}

func (a *App) SystemUninstall() error {
	return system.Uninstall()
}

func (a *App) SetLaunchAtLogin(enabled bool) error {
	return system.SetLaunchAtLogin(enabled)
}

func (a *App) SystemSetup() error {
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
