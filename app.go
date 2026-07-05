package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"orbit-app/internal/analyzer"
	"orbit-app/internal/config"
	"orbit-app/internal/database"
	"orbit-app/internal/projects"
	"orbit-app/internal/proxy"
	"orbit-app/internal/runtime"
	"orbit-app/internal/services"
	"orbit-app/internal/system"
	"orbit-app/internal/terminal"
	orbittls "orbit-app/internal/tls"

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
	services    *services.Manager
	svcStore    *services.Store
	svcConfig   *services.ConfigStore
	nodeAcq     *services.Acquirer
	phpAcq      *services.Acquirer
	runtimesCfg *services.RuntimesConfigStore
	terminals   *terminal.Manager
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

	sqlDB, queries, err := database.Open(ctx, cfg.DBPath)
	if err != nil {
		panic(fmt.Errorf("database: %w", err))
	}
	a.db = sqlDB

	repo := projects.NewRepository(sqlDB, queries)
	a.service = projects.NewService(repo, analyzer.New(), cfg.DomainSuffix)
	a.runtime = runtime.New(a.service, sqlDB, a.cfg)
	a.runtime.SetEmitter(runtime.NewWailsEmitter(ctx))
	a.terminals = terminal.NewManager()
	a.terminals.SetEmitter(runtime.NewWailsEmitter(ctx))
	a.registry = proxy.New(a.service, cfg.DomainSuffix)

	svcStore := services.NewStore(sqlDB)
	a.svcStore = svcStore
	svcBaseDir := filepath.Join(cfg.DataDir, "services")
	a.svcConfig = services.LoadConfig(cfg.DataDir)
	acq := services.NewAcquirer(svcStore, svcBaseDir)
	resolver := services.NewResolver(svcStore, services.AllowAll())
	a.services = services.NewManager(acq, resolver, a.svcConfig, filepath.Join(svcBaseDir, "data"))
	a.runtime.SetServices(a.services)
	a.nodeAcq = acq
	a.runtime.SetNodeAcquirer(acq)
	a.phpAcq = acq
	a.runtime.SetPHPAcquirer(acq)
	a.runtimesCfg = services.LoadRuntimesConfig(cfg.DataDir)
	a.runtime.SetRuntimesConfig(a.runtimesCfg)

	router := proxy.NewRouter(a.registry, a.runtime)
	recovery := proxy.NewRecoveryHandler(a.registry, a.runtime)

	svcEntries := map[string]proxy.ServiceRoute{}
	for _, e := range services.Catalog() {
		if e.WebDomain != "" && e.WebPort != 0 {
			svcEntries[e.WebDomain] = proxy.ServiceRoute{
				Engine:      e.ID,
				DisplayName: e.DisplayName,
				Upstream:    fmt.Sprintf("%s:%d", e.Bind, e.WebPort),
			}
		}
	}

	opts := proxy.Options{
		Addr:           cfg.ProxyAddr,
		Router:         router,
		Recovery:       recovery,
		Services:       proxy.NewServiceTable(cfg.DomainSuffix, svcEntries),
		ServiceStarter: a.services,
	}
	extraDomains := []string{}
	if domains, err := a.service.ListDomains(ctx); err == nil {
		for _, d := range domains {
			extraDomains = append(extraDomains, d.Domain)
		}
	}
	if mat, err := orbittls.Ensure(cfg.DomainSuffix, extraDomains); err == nil {
		opts.TLSAddr = cfg.ProxyTLSAddr
		opts.TLSCert = mat.LeafCert
		opts.TLSKey = mat.LeafKey
	} else {
		log.Printf("[tls] ensure: %v (internal tls listener disabled)", err)
	}
	a.proxyServer = proxy.NewServerWithOptions(opts)
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
	if a.services != nil {
		a.services.StopAll()
	}
	if a.terminals != nil {
		a.terminals.StopAll()
	}
	if a.runtime != nil {
		a.runtime.Close()
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

func (a *App) PathNeedsInstall(path string) bool {
	if path == "" {
		return false
	}
	st, err := os.Stat(filepath.Join(path, "node_modules"))
	return err != nil || !st.IsDir()
}

func (a *App) InstallProject(id string) error {
	return a.runtime.Install(a.ctx, id)
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
	for _, engine := range a.svcConfig.DefaultEngines() {
		_ = a.svcStore.SetEnabled(a.ctx, p.ID, engine, true)
	}
	return p, nil
}

func (a *App) ListProjects() ([]projects.Project, error) {
	return a.service.List(a.ctx)
}

func (a *App) GetProject(id string) (*projects.Project, error) {
	return a.service.Get(a.ctx, id)
}

func (a *App) SetProjectSecure(id string, secure bool) error {
	return a.service.SetSecure(a.ctx, id, secure)
}

func (a *App) DeleteProject(id string) error {
	a.runtime.RecordEvent(id, runtime.LevelWarn, runtime.SourceProject,
		"project deleted")
	a.terminals.Stop(id)
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

func (a *App) RuntimeMetrics(id string, sinceTs int64) []runtime.Sample {
	return a.runtime.Metrics(id, sinceTs)
}

func (a *App) RuntimeMetricsAll(sinceTs int64, ids []string) map[string][]runtime.Sample {
	return a.runtime.MetricsAll(sinceTs, ids)
}

func (a *App) SystemConfig() *config.Config {
	return a.cfg
}

func (a *App) SystemSaveConfig(cfg *config.Config) error {
	return a.cfg.Save()
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

func (a *App) OpenURL(url string) error {
	if url == "" {
		return fmt.Errorf("empty url")
	}
	wailsruntime.BrowserOpenURL(a.ctx, url)
	return nil
}

func (a *App) RevealInFinder(path string) error {
	if path == "" {
		return fmt.Errorf("empty path")
	}
	return exec.Command("open", path).Start()
}

func (a *App) OpenProject(id string) error {
	u, err := a.ProjectURL(id)
	if err != nil {
		return err
	}
	wailsruntime.BrowserOpenURL(a.ctx, u.URL)
	return nil
}

func (a *App) ListServices() []services.ServiceInfo {
	return a.services.List(a.ctx)
}

func (a *App) ServiceStatus(engine string) services.Snapshot {
	return a.services.Status(engine)
}

func (a *App) StartService(engine string) error {
	return a.services.StartManual(a.ctx, engine)
}

func (a *App) StopService(engine string) error {
	a.services.ForceStop(engine)
	return nil
}

func (a *App) UninstallService(engine string) error {
	return a.services.Uninstall(a.ctx, engine)
}

func (a *App) ServicesConfig() services.Config {
	return a.svcConfig.Get()
}

func (a *App) SaveServicesConfig(cfg services.Config) error {
	return a.svcConfig.Save(cfg)
}

func (a *App) ServicesDiskUsage() int64 {
	return a.services.DiskUsage()
}

func (a *App) ClearServicesData() error {
	return a.services.ClearData()
}

func (a *App) ListNodeVersions() []services.NodeVersionInfo {
	return services.NodeVersionsInfo(a.ctx, a.nodeAcq)
}

func (a *App) InstallNodeVersion(version string) error {
	return services.InstallNodeVersion(a.ctx, a.nodeAcq, version)
}

func (a *App) RemoveNodeVersion(version string) error {
	return services.RemoveNodeVersion(a.ctx, a.nodeAcq, version)
}

func (a *App) ListPHPVersions() []services.PHPVersionInfo {
	return services.PHPVersionsInfo(a.ctx, a.phpAcq)
}

func (a *App) InstallPHPVersion(version string) error {
	return services.InstallPHPVersion(a.ctx, a.phpAcq, version)
}

func (a *App) RemovePHPVersion(version string) error {
	return services.RemovePHPVersion(a.ctx, a.phpAcq, version)
}

type SystemRuntimeStatus struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
}

func (a *App) DetectSystemNode() SystemRuntimeStatus {
	sys, ok := services.DetectSystemNode()
	return SystemRuntimeStatus{Available: ok, Version: sys.Version}
}

func (a *App) DetectSystemPHP() SystemRuntimeStatus {
	sys, ok := services.DetectSystemPHP()
	return SystemRuntimeStatus{Available: ok, Version: sys.Version}
}

func (a *App) RuntimesConfig() services.RuntimesConfig {
	return a.runtimesCfg.Get()
}

func (a *App) SaveRuntimesConfig(cfg services.RuntimesConfig) error {
	return a.runtimesCfg.Save(cfg)
}

func (a *App) ServiceSetup(engine string) (services.SetupInfo, error) {
	e, ok := services.ResolveEngine(engine)
	if !ok {
		return services.SetupInfo{}, fmt.Errorf("unknown engine %q", engine)
	}
	return e.Setup, nil
}

func (a *App) TerminalStart(projectID string) error {
	proj, err := a.service.Get(a.ctx, projectID)
	if err != nil {
		return err
	}
	return a.terminals.Start(projectID, proj.Path)
}

func (a *App) TerminalWrite(projectID, data string) error {
	return a.terminals.Write(projectID, []byte(data))
}

func (a *App) TerminalResize(projectID string, cols, rows int) error {
	return a.terminals.Resize(projectID, cols, rows)
}

func (a *App) TerminalBuffer(projectID string) (string, error) {
	return string(a.terminals.Buffer(projectID)), nil
}

func (a *App) EnableServiceForProject(projectID, engine string) error {
	return a.svcStore.SetEnabled(a.ctx, projectID, engine, true)
}

func (a *App) DisableServiceForProject(projectID, engine string) error {
	return a.svcStore.SetEnabled(a.ctx, projectID, engine, false)
}

func (a *App) ProjectServices(projectID string) ([]string, error) {
	return a.svcStore.EnabledEngines(a.ctx, projectID)
}

func (a *App) DetectedServices(projectID string) ([]string, error) {
	p, err := a.service.Get(a.ctx, projectID)
	if err != nil {
		return nil, err
	}
	return services.DetectTokens(p.Path), nil
}

func (a *App) SystemStatus() system.Status {
	return system.Check()
}

func (a *App) SystemUninstall() error {
	return system.Uninstall()
}

func (a *App) TrustCA() error {
	return system.TrustCA()
}

func (a *App) UntrustCA() error {
	return system.UntrustCA()
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
