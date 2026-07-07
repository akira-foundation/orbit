package services

import (
	"context"
	"fmt"
	"path/filepath"
)

const pythonRelease = "20260623"

var pythonVersions = []struct {
	Major   string
	Version string
}{
	{"3.13", "3.13.14"},
	{"3.12", "3.12.13"},
	{"3.11", "3.11.15"},
}

var pythonTriples = map[string]string{
	"darwin/arm64": "aarch64-apple-darwin",
	"darwin/amd64": "x86_64-apple-darwin",
	"linux/arm64":  "aarch64-unknown-linux-gnu",
	"linux/amd64":  "x86_64-unknown-linux-gnu",
}

var pythonChecksums = map[string]string{
	"3.11.15/aarch64-apple-darwin":      "d2324bfd1a7b9fc44ccd884c3a2505bcab6691dbfd4f8270e10c50aaa4e19506",
	"3.11.15/aarch64-unknown-linux-gnu": "1de978b7039f345dacdddc3efb0726ce5b957bbbd34161037a4b426aabb18bf5",
	"3.11.15/x86_64-apple-darwin":       "38f3c18a4ccbd6faa09243c45c85d8e09b5a7b345e02f174346cf72ebf901f87",
	"3.11.15/x86_64-unknown-linux-gnu":  "60295e3e703b48c270e8d8c685195b8d5c2f0b8a596c1a910d7e24a2cc55afdd",
	"3.12.13/aarch64-apple-darwin":      "3724aa4dafb5f7b6c2cf98e89914e4248dc6bd2fe40407df4a2d73de99615f16",
	"3.12.13/aarch64-unknown-linux-gnu": "b14d074c43fdf03f01822fd07a15b3039eb0558503d1cb791791602cbe32908b",
	"3.12.13/x86_64-apple-darwin":       "7c57fdd1fa675190093700eb0d8e7117e1f9eae7c30a46dea5f8d5266bcfc791",
	"3.12.13/x86_64-unknown-linux-gnu":  "9fa869d69be54f6b8eeae64272fbd9bb0646e0e1a8da9d80e51ba5a3bee48930",
	"3.13.14/aarch64-apple-darwin":      "804c86c8665b18eb0df5070a79d828229018d145baea38a71a5c74c03f9b11d4",
	"3.13.14/aarch64-unknown-linux-gnu": "1199b22c83725a339ebaef36d39476d037fb7267187513090b6cc83bb4579477",
	"3.13.14/x86_64-apple-darwin":       "cd0023fb84de358d285c8e116cffd2f433086b943e752955dade521c12e78cab",
	"3.13.14/x86_64-unknown-linux-gnu":  "7fd02919461b368adafea3896ad082f5c4f759816d69681dcc6559bfbcd892af",
}

func PythonVersions() []string {
	out := make([]string, 0, len(pythonVersions))
	for _, v := range pythonVersions {
		out = append(out, v.Version)
	}
	return out
}

func PythonEngineForVersion(version string) (Engine, bool) {
	for _, e := range pythonEngines() {
		if e.Version == version {
			return e, true
		}
	}
	return Engine{}, false
}

type PythonVersionInfo struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Installed bool   `json:"installed"`
	DiskBytes int64  `json:"diskBytes"`
	Path      string `json:"path"`
}

func PythonVersionsInfo(ctx context.Context, acq *Acquirer) []PythonVersionInfo {
	out := make([]PythonVersionInfo, 0, len(pythonVersions))
	for _, e := range pythonEngines() {
		installed := acq.IsInstalled(ctx, e)
		dir := filepath.Join(acq.baseDir, e.ID, e.Version)
		var size int64
		var path string
		if installed {
			size = dirSize(dir)
			path = dir
		}
		out = append(out, PythonVersionInfo{
			ID:        e.ID,
			Version:   e.Version,
			Installed: installed,
			DiskBytes: size,
			Path:      path,
		})
	}
	return out
}

func InstallPythonVersion(ctx context.Context, acq *Acquirer, version string) error {
	e, ok := PythonEngineForVersion(version)
	if !ok {
		return fmt.Errorf("services: unknown python version %q", version)
	}
	_, err := acq.Ensure(ctx, e)
	return err
}

func RemovePythonVersion(ctx context.Context, acq *Acquirer, version string) error {
	e, ok := PythonEngineForVersion(version)
	if !ok {
		return fmt.Errorf("services: unknown python version %q", version)
	}
	return acq.Remove(ctx, e)
}

func pythonEngines() []Engine {
	out := make([]Engine, 0, len(pythonVersions))
	for _, v := range pythonVersions {
		v := v
		platforms := map[string]Platform{}
		for key, triple := range pythonTriples {
			platforms[key] = Platform{
				URL: fmt.Sprintf(
					"https://github.com/astral-sh/python-build-standalone/releases/download/%s/cpython-%s+%s-%s-install_only.tar.gz",
					pythonRelease, v.Version, pythonRelease, triple),
				SHA256:            pythonChecksums[v.Version+"/"+triple],
				ArchiveBinaryPath: "python/bin/python3",
			}
		}
		out = append(out, Engine{
			ID:          "python-" + v.Major,
			Version:     v.Version,
			DisplayName: "Python " + v.Major,
			Description: "Bundled CPython runtime used to run project dev commands (uvicorn/gunicorn/manage.py).",
			Family:      "python",
			ExtractTree: true,
			Platforms:   platforms,
		})
	}
	return out
}
