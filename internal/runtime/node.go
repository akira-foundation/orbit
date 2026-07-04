package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"orbit-app/internal/projects"
	"orbit-app/internal/services"
)

func (m *manager) SetNodeAcquirer(a *services.Acquirer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeAcquirer = a
}

func (m *manager) SetRuntimesConfig(c *services.RuntimesConfigStore) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runtimesConfig = c
}

func (m *manager) nodeBinDir(ctx context.Context, proj *projects.Project) (string, error) {
	// "", nil below means non-Node project, not a failure
	if proj.NodeVersion == "" {
		return "", nil
	}
	m.mu.RLock()
	acq := m.nodeAcquirer
	rtCfg := m.runtimesConfig
	m.mu.RUnlock()

	if rtCfg != nil && rtCfg.PreferSystemNode() {
		if sys, ok := services.DetectSystemNode(); ok {
			return sys.BinDir, nil
		}
	}

	if acq == nil {
		return "", nil
	}
	engine, ok := services.NodeEngineForVersion(proj.NodeVersion)
	if !ok {
		return "", fmt.Errorf("no bundled node engine for version %s", proj.NodeVersion)
	}
	binPath, err := acq.Ensure(ctx, engine)
	if err != nil {
		return "", fmt.Errorf("acquire node %s: %w", proj.NodeVersion, err)
	}
	return filepath.Dir(binPath), nil
}

func prependNodeBinDir(base []string, dir string) []string {
	if dir == "" {
		return base
	}
	for i, kv := range base {
		if strings.HasPrefix(kv, "PATH=") {
			base[i] = "PATH=" + dir + string(os.PathListSeparator) + kv[len("PATH="):]
			return base
		}
	}
	return append(base, "PATH="+dir)
}
