package share

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	localIPDomain = "local-ip.sh"
	localCertURL  = "https://local-ip.sh/server.pem"
	localKeyURL   = "https://local-ip.sh/server.key"
	renewWindow   = 7 * 24 * time.Hour
)

func LocalIPHost(ip string) string {
	return strings.ReplaceAll(ip, ".", "-") + "." + localIPDomain
}

func EnsureLocalCert(dir string) (certPath, keyPath string, err error) {
	certPath = filepath.Join(dir, "local-ip.pem")
	keyPath = filepath.Join(dir, "local-ip.key")
	if certFresh(certPath) && fileExists(keyPath) {
		return certPath, keyPath, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	if err := download(localCertURL, certPath); err != nil {
		return "", "", fmt.Errorf("share: fetch cert: %w", err)
	}
	if err := download(localKeyURL, keyPath); err != nil {
		return "", "", fmt.Errorf("share: fetch key: %w", err)
	}
	return certPath, keyPath, nil
}

func certFresh(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	return time.Until(cert.NotAfter) > renewWindow
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func download(url, dest string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
