package bindings

import (
	"context"

	"orbit-app/internal/services"
)

type SystemRuntimeStatus struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
}

type Versions struct {
	ctx         context.Context
	nodeAcq     *services.Acquirer
	phpAcq      *services.Acquirer
	pythonAcq   *services.Acquirer
	runtimesCfg *services.RuntimesConfigStore
}

func NewVersions() *Versions { return &Versions{} }

func (v *Versions) Attach(d Deps) {
	v.ctx = d.Ctx
	v.nodeAcq = d.NodeAcq
	v.phpAcq = d.PHPAcq
	v.pythonAcq = d.PythonAcq
	v.runtimesCfg = d.RuntimesCfg
}

func (v *Versions) ListNodeVersions() []services.NodeVersionInfo {
	return services.NodeVersionsInfo(v.ctx, v.nodeAcq)
}

func (v *Versions) InstallNodeVersion(version string) error {
	return services.InstallNodeVersion(v.ctx, v.nodeAcq, version)
}

func (v *Versions) RemoveNodeVersion(version string) error {
	return services.RemoveNodeVersion(v.ctx, v.nodeAcq, version)
}

func (v *Versions) ListPHPVersions() []services.PHPVersionInfo {
	return services.PHPVersionsInfo(v.ctx, v.phpAcq)
}

func (v *Versions) InstallPHPVersion(version string) error {
	return services.InstallPHPVersion(v.ctx, v.phpAcq, version)
}

func (v *Versions) RemovePHPVersion(version string) error {
	return services.RemovePHPVersion(v.ctx, v.phpAcq, version)
}

func (v *Versions) ListPythonVersions() []services.PythonVersionInfo {
	return services.PythonVersionsInfo(v.ctx, v.pythonAcq)
}

func (v *Versions) InstallPythonVersion(version string) error {
	return services.InstallPythonVersion(v.ctx, v.pythonAcq, version)
}

func (v *Versions) RemovePythonVersion(version string) error {
	return services.RemovePythonVersion(v.ctx, v.pythonAcq, version)
}

func (v *Versions) DetectSystemPython() SystemRuntimeStatus {
	sys, ok := services.DetectSystemPython()
	return SystemRuntimeStatus{Available: ok, Version: sys.Version}
}

func (v *Versions) DetectSystemNode() SystemRuntimeStatus {
	sys, ok := services.DetectSystemNode()
	return SystemRuntimeStatus{Available: ok, Version: sys.Version}
}

func (v *Versions) DetectSystemPHP() SystemRuntimeStatus {
	sys, ok := services.DetectSystemPHP()
	return SystemRuntimeStatus{Available: ok, Version: sys.Version}
}

func (v *Versions) DetectSystemDocker() SystemRuntimeStatus {
	sys, ok := services.DetectSystemDocker()
	return SystemRuntimeStatus{Available: ok, Version: sys.Version}
}

func (v *Versions) RuntimesConfig() services.RuntimesConfig {
	return v.runtimesCfg.Get()
}

func (v *Versions) SaveRuntimesConfig(cfg services.RuntimesConfig) error {
	return v.runtimesCfg.Save(cfg)
}
