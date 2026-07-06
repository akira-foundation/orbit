package bindings

import (
	"context"
	"fmt"

	"orbit-app/internal/projects"
	"orbit-app/internal/services"
)

type Services struct {
	ctx       context.Context
	services  *services.Manager
	svcStore  *services.Store
	svcConfig *services.ConfigStore
	service   *projects.Service
}

func NewServices() *Services { return &Services{} }

func (s *Services) Attach(d Deps) {
	s.ctx = d.Ctx
	s.services = d.Services
	s.svcStore = d.SvcStore
	s.svcConfig = d.SvcConfig
	s.service = d.Service
}

func (s *Services) ListServices() []services.ServiceInfo {
	return s.services.List(s.ctx)
}

func (s *Services) ServiceStatus(engine string) services.Snapshot {
	return s.services.Status(engine)
}

func (s *Services) StartService(engine string) error {
	return s.services.StartManual(s.ctx, engine)
}

func (s *Services) StopService(engine string) error {
	s.services.ForceStop(engine)
	return nil
}

func (s *Services) UninstallService(engine string) error {
	return s.services.Uninstall(s.ctx, engine)
}

func (s *Services) ServicesConfig() services.Config {
	return s.svcConfig.Get()
}

func (s *Services) SaveServicesConfig(cfg services.Config) error {
	return s.svcConfig.Save(cfg)
}

func (s *Services) ServicesDiskUsage() int64 {
	return s.services.DiskUsage()
}

func (s *Services) ClearServicesData() error {
	return s.services.ClearData()
}

func (s *Services) ServiceSetup(engine string) (services.SetupInfo, error) {
	e, ok := services.ResolveEngine(engine)
	if !ok {
		return services.SetupInfo{}, fmt.Errorf("unknown engine %q", engine)
	}
	return e.Setup, nil
}

func (s *Services) EnableServiceForProject(projectID, engine string) error {
	return s.svcStore.SetEnabled(s.ctx, projectID, engine, true)
}

func (s *Services) DisableServiceForProject(projectID, engine string) error {
	return s.svcStore.SetEnabled(s.ctx, projectID, engine, false)
}

func (s *Services) ProjectServices(projectID string) ([]string, error) {
	return s.svcStore.EnabledEngines(s.ctx, projectID)
}

func (s *Services) DetectedServices(projectID string) ([]string, error) {
	p, err := s.service.Get(s.ctx, projectID)
	if err != nil {
		return nil, err
	}
	return services.DetectTokens(p.Path), nil
}
