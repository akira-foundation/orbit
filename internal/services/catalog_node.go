package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

var nodeVersions = []struct {
	Major   string
	Version string
}{
	{"24", "24.18.0"},
	{"22", "22.23.1"},
	{"20", "20.20.2"},
}

var nodeTriples = map[string]string{
	"darwin/arm64": "darwin-arm64",
	"darwin/amd64": "darwin-x64",
	"linux/arm64":  "linux-arm64",
	"linux/amd64":  "linux-x64",
}

var nodeChecksums = map[string]string{
	"24.18.0/darwin-arm64": "e1a97e14c99c803e96c7339403282ea05a499c32f8d83defe9ef5ec66f979ed1",
	"24.18.0/darwin-x64":   "dfd0dbd3e721503434df7b7205e719f61b3a3a31b2bcf9729b8b91fea240f080",
	"24.18.0/linux-arm64":  "6b4484c2190274175df9aa8f28e2d758a819cb1c1fe6ab481e2f95b463ab8508",
	"24.18.0/linux-x64":    "783130984963db7ba9cbd01089eaf2c2efb055c7c1693c943174b967b3050cb8",
	"22.23.1/darwin-arm64": "ef28d8fab2c0e4314522d4bb1b7173270aa3937e93b92cb7de79c112ac1fa953",
	"22.23.1/darwin-x64":   "b8da981b8a0b1241b70249204916da76c63573ddf5814dbd2d1e41069105cb81",
	"22.23.1/linux-arm64":  "543fa39e57d4c07855939459a323f4deb9a79dd1bb45e6e99458b0f2de10db8d",
	"22.23.1/linux-x64":    "7a8cb04b4a1df4eaf432125324b81b29a088e73570a23259a8de1c65d07fc129",
	"20.20.2/darwin-arm64": "466e05f3477c20dfb723054dfebffe55bc74660ee77f612166fca121dacb65b6",
	"20.20.2/darwin-x64":   "8be6f5e4bb128c82774f8a0b8d7a1cc1365a7977d9657cece0ca647b3fe04e61",
	"20.20.2/linux-arm64":  "47ef73d543ecf6eb19435f6c03a0ac4809b3bf0dd6b26c7c571efc2a6572a74d",
	"20.20.2/linux-x64":    "19e56f0825510207dd904f087fe52faa0a4eb6b2aab5f0ea7a33830d04888b8b",
}

func NodeVersions() []string {
	out := make([]string, 0, len(nodeVersions))
	for _, v := range nodeVersions {
		out = append(out, v.Version)
	}
	return out
}

func NodeEngineForVersion(version string) (Engine, bool) {
	for _, e := range nodeEngines() {
		if e.Version == version {
			return e, true
		}
	}
	return Engine{}, false
}

type NodeVersionInfo struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	DiskBytes int64  `json:"diskBytes"`
	Path      string `json:"path"`
}

func NodeVersionsInfo(ctx context.Context, acq *Acquirer) []NodeVersionInfo {
	out := make([]NodeVersionInfo, 0, len(nodeVersions))
	for _, e := range nodeEngines() {
		installed := acq.IsInstalled(ctx, e)
		dir := filepath.Join(acq.baseDir, e.ID, e.Version)
		var size int64
		var path string
		if installed {
			size = dirSize(dir)
			path = dir
		}
		out = append(out, NodeVersionInfo{
			ID:        e.ID,
			Version:   e.Version,
			Installed: installed,
			DiskBytes: size,
			Path:      path,
		})
	}
	return out
}

func InstallNodeVersion(ctx context.Context, acq *Acquirer, version string) error {
	e, ok := NodeEngineForVersion(version)
	if !ok {
		return fmt.Errorf("services: unknown node version %q", version)
	}
	_, err := acq.Ensure(ctx, e)
	return err
}

// safe during active use: the OS keeps a running process's already-open binary alive
func RemoveNodeVersion(ctx context.Context, acq *Acquirer, version string) error {
	e, ok := NodeEngineForVersion(version)
	if !ok {
		return fmt.Errorf("services: unknown node version %q", version)
	}
	return acq.Remove(ctx, e)
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.Walk(root, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// unlike Postgres/Mailpit, never spawned as a process by services.Manager: only Ensure()/binPath() are reused
func nodeEngines() []Engine {
	out := make([]Engine, 0, len(nodeVersions))
	for _, v := range nodeVersions {
		v := v
		platforms := map[string]Platform{}
		for key, triple := range nodeTriples {
			root := fmt.Sprintf("node-v%s-%s", v.Version, triple)
			platforms[key] = Platform{
				URL: fmt.Sprintf(
					"https://nodejs.org/dist/v%s/node-v%s-%s.tar.gz",
					v.Version, v.Version, triple),
				SHA256:            nodeChecksums[v.Version+"/"+triple],
				ArchiveBinaryPath: root + "/bin/node",
			}
		}
		out = append(out, Engine{
			ID:          "node-" + v.Major,
			Version:     v.Version,
			DisplayName: "Node.js " + v.Major,
			Description: "Bundled Node.js runtime used to run project dev commands (npm/npx/vite).",
			Family:      "node",
			ExtractTree: true,
			Platforms:   platforms,
		})
	}
	return out
}
