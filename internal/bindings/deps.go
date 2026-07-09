package bindings

import (
	"context"

	"orbit-app/internal/config"
	"orbit-app/internal/parked"
	"orbit-app/internal/projects"
	"orbit-app/internal/proxy"
	"orbit-app/internal/runtime"
	"orbit-app/internal/services"
	"orbit-app/internal/share"
	"orbit-app/internal/terminal"
)

type Deps struct {
	Ctx           context.Context
	Cfg           *config.Config
	Service       *projects.Service
	Runtime       runtime.Manager
	Services      *services.Manager
	SvcStore      *services.Store
	SvcConfig     *services.ConfigStore
	NodeAcq       *services.Acquirer
	PHPAcq        *services.Acquirer
	PythonAcq     *services.Acquirer
	RuntimesCfg   *services.RuntimesConfigStore
	Terminals     *terminal.Manager
	ProxyServer   *proxy.Server
	Share         *share.State
	ParkedStore   *parked.Store
	ParkedManager *parked.Manager
}
