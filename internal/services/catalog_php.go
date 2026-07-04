package services

import (
	"context"
	"fmt"
	"path/filepath"
)

var phpVersions = []struct {
	Minor   string
	Version string
}{
	{"8.5", "8.5.8"},
	{"8.4", "8.4.23"},
	{"8.3", "8.3.32"},
}

// TODO(@kid): URL is a placeholder pending a hosting decision (GitHub Releases vs DO Spaces).
var phpChecksums = map[string]string{
	"8.5.8/darwin-arm64":  "4fe33bf161e079cbb8f6b9d355062f3d65ca455174242b2b93fb0b65a027f34b",
	"8.4.23/darwin-arm64": "75229afdf54bc89e609055238c1a5f54ea78b6a716ee1a54a0f7bb441a652432",
	"8.3.32/darwin-arm64": "a6946dcfe9578a0d979ebc68640b54f875554a2889181f0009b38cf2a51690a5",
}

func phpEngines() []Engine {
	out := make([]Engine, 0, len(phpVersions))
	for _, v := range phpVersions {
		v := v
		platforms := map[string]Platform{}
		if sum, ok := phpChecksums[v.Version+"/darwin-arm64"]; ok {
			root := fmt.Sprintf("php-%s-darwin-arm64", v.Version)
			platforms["darwin/arm64"] = Platform{
				URL:               fmt.Sprintf("https://TODO-orbit-releases/php/%s/%s.tar.gz", v.Version, root),
				SHA256:            sum,
				ArchiveBinaryPath: root + "/bin/php-fpm",
			}
		}
		out = append(out, Engine{
			ID:          "php-" + v.Minor,
			Version:     v.Version,
			DisplayName: "PHP " + v.Minor,
			Description: "Bundled PHP runtime with php-fpm, used to run Laravel projects.",
			Family:      "php",
			ExtractTree: true,
			Platforms:   platforms,
		})
	}
	return out
}

func PHPVersions() []string {
	out := make([]string, 0, len(phpVersions))
	for _, v := range phpVersions {
		out = append(out, v.Version)
	}
	return out
}

func PHPEngineForVersion(version string) (Engine, bool) {
	for _, e := range phpEngines() {
		if e.Version == version {
			return e, true
		}
	}
	return Engine{}, false
}

type PHPVersionInfo struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	DiskBytes int64  `json:"diskBytes"`
	Path      string `json:"path"`
}

func PHPVersionsInfo(ctx context.Context, acq *Acquirer) []PHPVersionInfo {
	out := make([]PHPVersionInfo, 0, len(phpVersions))
	for _, e := range phpEngines() {
		installed := acq.IsInstalled(ctx, e)
		dir := filepath.Join(acq.baseDir, e.ID, e.Version)
		var size int64
		var path string
		if installed {
			size = dirSize(dir)
			path = dir
		}
		out = append(out, PHPVersionInfo{
			ID:        e.ID,
			Version:   e.Version,
			Installed: installed,
			DiskBytes: size,
			Path:      path,
		})
	}
	return out
}

func InstallPHPVersion(ctx context.Context, acq *Acquirer, version string) error {
	e, ok := PHPEngineForVersion(version)
	if !ok {
		return fmt.Errorf("services: unknown php version %q", version)
	}
	_, err := acq.Ensure(ctx, e)
	return err
}

func RemovePHPVersion(ctx context.Context, acq *Acquirer, version string) error {
	e, ok := PHPEngineForVersion(version)
	if !ok {
		return fmt.Errorf("services: unknown php version %q", version)
	}
	return acq.Remove(ctx, e)
}
