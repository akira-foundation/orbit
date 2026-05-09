package main

import (
	"context"
	"database/sql"
	"fmt"

	"orbit-app/internal/analyzer"
	"orbit-app/internal/config"
	"orbit-app/internal/database"
	"orbit-app/internal/projects"
	"orbit-app/internal/proxy"
	"orbit-app/internal/runtime"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx     context.Context
	cfg     *config.Config
	db      *sql.DB
	service *projects.Service
	runtime runtime.Manager
	proxy   proxy.Manager
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
	a.service = projects.NewService(repo, analyzer.New(), cfg.DomainTLD)
	a.runtime = runtime.NewStub()
	a.proxy = proxy.NewStub()
}

func (a *App) shutdown(_ context.Context) {
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
		SuggestedDomain: slug + "." + a.cfg.DomainTLD,
	}, nil
}

func (a *App) AddProject(path string) (*projects.Project, error) {
	return a.service.Create(a.ctx, path)
}

func (a *App) ListProjects() ([]projects.Project, error) {
	return a.service.List(a.ctx)
}

func (a *App) GetProject(id string) (*projects.Project, error) {
	return a.service.Get(a.ctx, id)
}

func (a *App) DeleteProject(id string) error {
	return a.service.Delete(a.ctx, id)
}

func (a *App) StartProject(id string) error {
	if err := a.runtime.Start(a.ctx, id); err != nil {
		return err
	}
	return a.service.UpdateStatus(a.ctx, id, projects.StatusRunning)
}

func (a *App) StopProject(id string) error {
	if err := a.runtime.Stop(a.ctx, id); err != nil {
		return err
	}
	return a.service.UpdateStatus(a.ctx, id, projects.StatusStopped)
}

func (a *App) SelectProjectFolder() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "Select project folder",
	})
}
