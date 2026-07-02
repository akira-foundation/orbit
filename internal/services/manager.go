package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type instance struct {
	engine    string
	proc      *svcProcess
	status    string
	refs      map[string]struct{}
	startedAt time.Time
}

type Manager struct {
	mu        sync.Mutex
	acq       *Acquirer
	resolver  *Resolver
	cfg       *ConfigStore
	dataDir   string
	instances map[string]*instance
	held      map[string][]string

	starter  func(bin string, args, env []string) (*svcProcess, error)
	ensure   func(ctx context.Context, e Engine) (string, error)
	dialAddr func(e Engine) string
	reap     func(engine string)
	now      func() time.Time
}

func NewManager(acq *Acquirer, resolver *Resolver, cfg *ConfigStore, dataDir string) *Manager {
	m := &Manager{
		acq:       acq,
		resolver:  resolver,
		cfg:       cfg,
		dataDir:   dataDir,
		instances: map[string]*instance{},
		held:      map[string][]string{},
	}
	m.starter = spawnService
	m.ensure = acq.Ensure
	m.dialAddr = func(e Engine) string {
		return fmt.Sprintf("%s:%d", e.Bind, e.Port)
	}
	m.reap = func(engine string) {
		if e, ok := ResolveEngine(engine); ok {
			reapByPort(e.Bind, e.Port)
			reapByPort(e.Bind, e.WebPort)
			reapByPort(e.Bind, e.SMTPPort)
			reapByPort(e.Bind, e.APIPort)
		}
	}
	m.now = time.Now
	go m.idleSweeper()
	return m
}

func (m *Manager) Acquire(ctx context.Context, engine, projectID string) error {
	e, ok := ResolveEngine(engine)
	if !ok {
		return fmt.Errorf("services: unknown engine %q", engine)
	}

	m.mu.Lock()
	inst, running := m.instances[engine]
	if running {
		inst.refs[projectID] = struct{}{}
		m.mu.Unlock()
		return nil
	}
	m.mu.Unlock()

	if m.reachable(e) {
		m.mu.Lock()
		m.instances[engine] = &instance{
			engine:    engine,
			status:    "running",
			refs:      map[string]struct{}{projectID: {}},
			startedAt: m.now(),
		}
		m.mu.Unlock()
		log.Printf("[services] %s adopted (already running)", engine)
		return nil
	}

	m.mu.Lock()
	inst = &instance{engine: engine, status: "starting", refs: map[string]struct{}{projectID: {}}}
	m.instances[engine] = inst
	m.mu.Unlock()

	bin, err := m.ensure(ctx, e)
	if err != nil {
		m.fail(engine)
		return err
	}

	dataDir := m.dataDir + "/" + engine
	if err := ensureDir(dataDir); err != nil {
		m.fail(engine)
		return err
	}

	proc, err := m.starter(bin, e.Args(platformOrEmpty(e), dataDir), nil)
	if err != nil {
		m.fail(engine)
		return err
	}

	m.mu.Lock()
	inst.proc = proc
	m.mu.Unlock()

	go m.pipeLogs(engine, proc)

	if err := waitDialable(ctx, m.dialAddr(e), 15*time.Second); err != nil {
		_ = proc.forceKill()
		m.fail(engine)
		return err
	}

	m.mu.Lock()
	inst.status = "running"
	inst.startedAt = m.now()
	m.mu.Unlock()
	log.Printf("[services] %s running pid=%d", engine, proc.cmd.Process.Pid)
	return nil
}

func (m *Manager) Release(engine, projectID string) {
	m.mu.Lock()
	inst, ok := m.instances[engine]
	if !ok {
		m.mu.Unlock()
		return
	}
	delete(inst.refs, projectID)
	if len(inst.refs) > 0 {
		m.mu.Unlock()
		return
	}
	proc := inst.proc
	delete(m.instances, engine)
	m.mu.Unlock()

	m.terminate(engine, proc)
	log.Printf("[services] %s stopped (refcount zero)", engine)
}

func (m *Manager) terminate(engine string, proc *svcProcess) {
	if proc != nil {
		_ = proc.kill()
		go func() {
			time.Sleep(5 * time.Second)
			_ = proc.forceKill()
		}()
	}
	m.reap(engine)
}

func (m *Manager) reachable(e Engine) bool {
	if e.Port == 0 {
		return false
	}
	return dialOnce(m.dialAddr(e))
}

func (m *Manager) OnProjectStart(ctx context.Context, projectID, projectPath, slug string) (map[string]string, error) {
	if m.cfg != nil && !m.cfg.AutoManage() {
		return nil, nil
	}
	engines, err := m.resolver.EnabledFor(ctx, projectID, projectPath)
	if err != nil {
		return nil, err
	}
	creds := map[string]string{}
	var acquired []string
	for _, engine := range engines {
		if err := m.Acquire(ctx, engine, projectID); err != nil {
			log.Printf("[services] acquire %s for %s: %v", engine, projectID, err)
			continue
		}
		acquired = append(acquired, engine)
		for k, v := range m.provisionEngine(ctx, engine, slug) {
			creds[k] = v
		}
	}
	m.mu.Lock()
	m.held[projectID] = acquired
	m.mu.Unlock()
	return creds, nil
}

func (m *Manager) provisionEngine(ctx context.Context, engine, slug string) map[string]string {
	e, ok := ResolveEngine(engine)
	if !ok || e.Provision == nil {
		return nil
	}
	env, err := e.Provision(ctx, e.Port, slug)
	if err != nil {
		log.Printf("[services] provision %s for %s: %v", engine, slug, err)
		return nil
	}
	return env
}

func (m *Manager) OnProjectStop(projectID string) {
	m.mu.Lock()
	engines := m.held[projectID]
	delete(m.held, projectID)
	m.mu.Unlock()
	for _, e := range engines {
		m.Release(e, projectID)
	}
}

func (m *Manager) StartManual(ctx context.Context, engine string) error {
	return m.Acquire(ctx, engine, "__manual__")
}

func (m *Manager) ForceStop(engine string) {
	m.mu.Lock()
	inst, ok := m.instances[engine]
	var proc *svcProcess
	if ok {
		proc = inst.proc
		delete(m.instances, engine)
		for pid, engines := range m.held {
			m.held[pid] = removeString(engines, engine)
		}
	}
	m.mu.Unlock()

	m.terminate(engine, proc)
	log.Printf("[services] %s force-stopped", engine)
}

func (m *Manager) Uninstall(ctx context.Context, engine string) error {
	e, ok := ResolveEngine(engine)
	if !ok {
		return fmt.Errorf("services: unknown engine %q", engine)
	}
	m.ForceStop(engine)
	if err := m.acq.Remove(ctx, e); err != nil {
		return err
	}
	return os.RemoveAll(m.dataDir + "/" + engine)
}

func removeString(xs []string, v string) []string {
	out := xs[:0]
	for _, x := range xs {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}

func (m *Manager) Status(engine string) Snapshot {
	m.mu.Lock()
	inst, ok := m.instances[engine]
	if ok {
		pid := 0
		if inst.proc != nil && inst.proc.cmd.Process != nil {
			pid = inst.proc.cmd.Process.Pid
		}
		snap := Snapshot{Engine: engine, Status: inst.status, PID: pid, Refs: len(inst.refs)}
		m.mu.Unlock()
		return snap
	}
	m.mu.Unlock()

	if e, found := ResolveEngine(engine); found && m.reachable(e) {
		return Snapshot{Engine: engine, Status: "running"}
	}
	return Snapshot{Engine: engine, Status: "stopped"}
}

func (m *Manager) List(ctx context.Context) []ServiceInfo {
	out := make([]ServiceInfo, 0)
	for _, e := range Catalog() {
		snap := m.Status(e.ID)
		out = append(out, ServiceInfo{
			Engine:      e.ID,
			DisplayName: e.DisplayName,
			Description: e.Description,
			Version:     e.Version,
			Status:      snap.Status,
			WebURL:      e.WebDomain + ".orbit.test",
			Installed:   m.acq.IsInstalled(ctx, e),
			Refs:        snap.Refs,
		})
	}
	return out
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	procs := make([]*svcProcess, 0, len(m.instances))
	for _, inst := range m.instances {
		if inst.proc != nil {
			procs = append(procs, inst.proc)
		}
	}
	m.instances = map[string]*instance{}
	m.held = map[string][]string{}
	m.mu.Unlock()
	for _, p := range procs {
		_ = p.forceKill()
	}
}

func (m *Manager) idleSweeper() {
	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for range tick.C {
		if m.cfg == nil {
			continue
		}
		mins := m.cfg.IdleStopMinutes()
		if mins <= 0 {
			continue
		}
		cutoff := m.now().Add(-time.Duration(mins) * time.Minute)
		var stop []string
		m.mu.Lock()
		for engine, inst := range m.instances {
			if inst.status != "running" || !inst.startedAt.Before(cutoff) {
				continue
			}
			if hasProjectRef(inst.refs) {
				continue
			}
			stop = append(stop, engine)
		}
		m.mu.Unlock()
		for _, engine := range stop {
			log.Printf("[services] %s idle-stop after %dm", engine, mins)
			m.ForceStop(engine)
		}
	}
}

func hasProjectRef(refs map[string]struct{}) bool {
	for r := range refs {
		if r != "__manual__" {
			return true
		}
	}
	return false
}

func (m *Manager) DiskUsage() int64 {
	var total int64
	for _, root := range []string{m.acq.baseDir, m.dataDir} {
		_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				total += info.Size()
			}
			return nil
		})
	}
	return total
}

func (m *Manager) ClearData() error {
	for _, e := range Catalog() {
		m.ForceStop(e.ID)
	}
	entries, err := os.ReadDir(m.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(m.dataDir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) fail(engine string) {
	m.mu.Lock()
	if inst, ok := m.instances[engine]; ok {
		inst.status = "error"
		if inst.proc != nil {
			_ = inst.proc.forceKill()
		}
		delete(m.instances, engine)
	}
	m.mu.Unlock()
}

func (m *Manager) pipeLogs(engine string, proc *svcProcess) {
	if proc.stdout == nil {
		return
	}
	scanServiceLines(proc.stdout, func(line string) bool {
		log.Printf("[services:%s] %s", engine, line)
		return true
	})
}

type ServiceInfo struct {
	Engine      string `json:"engine"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	WebURL      string `json:"webUrl"`
	Installed   bool   `json:"installed"`
	Refs        int    `json:"refs"`
}

type Snapshot struct {
	Engine string `json:"engine"`
	Status string `json:"status"`
	PID    int    `json:"pid"`
	Refs   int    `json:"refs"`
}

func platformOrEmpty(e Engine) Platform {
	if p, ok := e.CurrentPlatform(); ok {
		return p
	}
	return Platform{ArchiveBinaryPath: e.ID}
}
