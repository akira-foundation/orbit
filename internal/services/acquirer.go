package services

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Acquirer struct {
	store   *Store
	baseDir string
	client  *http.Client
}

func NewAcquirer(store *Store, baseDir string) *Acquirer {
	return &Acquirer{store: store, baseDir: baseDir, client: http.DefaultClient}
}

func (a *Acquirer) binPath(e Engine, p Platform) string {
	return filepath.Join(a.baseDir, e.ID, e.Version, filepath.Base(p.ArchiveBinaryPath))
}

func (a *Acquirer) IsInstalled(ctx context.Context, e Engine) bool {
	p, ok := e.CurrentPlatform()
	if !ok {
		return false
	}
	if st, err := os.Stat(a.binPath(e, p)); err == nil && !st.IsDir() {
		return true
	}
	_, found, _ := a.store.Binary(ctx, e.ID, e.Version)
	return found
}

func (a *Acquirer) Ensure(ctx context.Context, e Engine) (string, error) {
	p, ok := e.CurrentPlatform()
	if !ok {
		return "", fmt.Errorf("services: %s has no build for %s", e.ID, e.PlatformKey())
	}
	dst := a.binPath(e, p)
	if st, err := os.Stat(dst); err == nil && !st.IsDir() {
		return dst, nil
	}

	if p.SHA256 == "" {
		return "", fmt.Errorf("services: %s has no pinned checksum for %s", e.ID, e.PlatformKey())
	}

	archive, err := a.fetch(ctx, p.URL)
	if err != nil {
		return "", fmt.Errorf("services: download %s: %w", e.ID, err)
	}

	got := sha256.Sum256(archive)
	gotHex := hex.EncodeToString(got[:])
	if !strings.EqualFold(gotHex, p.SHA256) {
		return "", fmt.Errorf("services: %s checksum mismatch: got %s want %s", e.ID, gotHex, p.SHA256)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return "", err
	}
	if err := extractFromTarGz(archive, p.ArchiveBinaryPath, dst); err != nil {
		return "", fmt.Errorf("services: extract %s: %w", e.ID, err)
	}
	if err := os.Chmod(dst, 0o755); err != nil {
		return "", err
	}
	if err := a.store.RecordBinary(ctx, e.ID, e.Version, dst, gotHex); err != nil {
		return "", err
	}
	return dst, nil
}

func (a *Acquirer) fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

func extractFromTarGz(archive []byte, wantPath, dst string) error {
	gz, err := gzip.NewReader(strings.NewReader(string(archive)))
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s not found in archive", wantPath)
		}
		if err != nil {
			return err
		}
		if path.Clean(hdr.Name) != path.Clean(wantPath) {
			continue
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, tr)
		return err
	}
}

func (a *Acquirer) Remove(ctx context.Context, e Engine) error {
	if err := os.RemoveAll(filepath.Join(a.baseDir, e.ID)); err != nil {
		return err
	}
	return a.store.DeleteBinaries(ctx, e.ID)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
