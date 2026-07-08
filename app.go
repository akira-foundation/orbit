package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"time"

	"orbit-app/internal/analyzer"
	"orbit-app/internal/bindings"
	"orbit-app/internal/config"
	"orbit-app/internal/database"
	"orbit-app/internal/projects"
	"orbit-app/internal/proxy"
	"orbit-app/internal/runtime"
	"orbit-app/internal/services"
	"orbit-app/internal/share"
	"orbit-app/internal/terminal"
	orbittls "orbit-app/internal/tls"
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
	pythonAcq   *services.Acquirer
	runtimesCfg *services.RuntimesConfigStore
	terminals   *terminal.Manager

	projectsAPI *bindings.Projects
	groupsAPI   *bindings.Groups
	runtimeAPI  *bindings.Runtime
	servicesAPI *bindings.Services
	mailAPI     *bindings.Mail
	versionsAPI *bindings.Versions
	terminalAPI *bindings.Terminal
	systemAPI   *bindings.System
	dataAPI     *bindings.Data
	shareAPI    *bindings.Share
	requestsAPI *bindings.Requests
	composeAPI  *bindings.Compose
	copilotAPI  *bindings.Copilot
	share       *share.State
}

func NewApp() *App {
	return &App{
		projectsAPI: bindings.NewProjects(),
		groupsAPI:   bindings.NewGroups(),
		runtimeAPI:  bindings.NewRuntime(),
		servicesAPI: bindings.NewServices(),
		mailAPI:     bindings.NewMail(),
		versionsAPI: bindings.NewVersions(),
		terminalAPI: bindings.NewTerminal(),
		systemAPI:   bindings.NewSystem(),
		dataAPI:     bindings.NewData(),
		shareAPI:    bindings.NewShare(),
		requestsAPI: bindings.NewRequests(),
		composeAPI:  bindings.NewCompose(),
		copilotAPI:  bindings.NewCopilot(),
		share:       share.NewState(),
	}
}

func (a *App) boundAPIs() []interface{} {
	return []interface{}{
		a.projectsAPI,
		a.groupsAPI,
		a.runtimeAPI,
		a.servicesAPI,
		a.mailAPI,
		a.versionsAPI,
		a.terminalAPI,
		a.systemAPI,
		a.dataAPI,
		a.shareAPI,
		a.requestsAPI,
		a.composeAPI,
		a.copilotAPI,
	}
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
	a.pythonAcq = acq
	a.runtime.SetPythonAcquirer(acq)
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
		Suffix:         cfg.DomainSuffix,
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
	var extraIPs []net.IP
	if lanIP, err := share.PrimaryLANIP(); err == nil {
		if ip := net.ParseIP(lanIP); ip != nil {
			extraIPs = append(extraIPs, ip)
		}
	}
	if mat, err := orbittls.Ensure(cfg.DomainSuffix, extraDomains, extraIPs); err == nil {
		opts.TLSAddr = cfg.ProxyTLSAddr
		opts.TLSCert = mat.LeafCert
		opts.TLSKey = mat.LeafKey
		go orbittls.StartSweep(ctx, mat.Dir, runtime.NewWailsEmitter(ctx).Emit)
	} else {
		log.Printf("[tls] ensure: %v (internal tls listener disabled)", err)
	}
	a.proxyServer = proxy.NewServerWithOptions(opts)
	go func() {
		if err := a.proxyServer.ListenAndServe(); err != nil {
			log.Printf("[proxy] server error: %v", err)
		}
	}()

	deps := bindings.Deps{
		Ctx:         ctx,
		Cfg:         a.cfg,
		Service:     a.service,
		Runtime:     a.runtime,
		Services:    a.services,
		SvcStore:    a.svcStore,
		SvcConfig:   a.svcConfig,
		NodeAcq:     a.nodeAcq,
		PHPAcq:      a.phpAcq,
		PythonAcq:   a.pythonAcq,
		RuntimesCfg: a.runtimesCfg,
		Terminals:   a.terminals,
		ProxyServer: a.proxyServer,
		Share:       a.share,
	}
	for _, api := range []interface{ Attach(bindings.Deps) }{
		a.projectsAPI, a.groupsAPI, a.runtimeAPI, a.servicesAPI,
		a.mailAPI, a.versionsAPI, a.terminalAPI, a.systemAPI,
		a.dataAPI, a.shareAPI, a.requestsAPI, a.composeAPI, a.copilotAPI,
	} {
		api.Attach(deps)
	}

	go a.mailAPI.Watch(ctx)
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
