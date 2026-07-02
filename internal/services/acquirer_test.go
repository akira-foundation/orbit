package services

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestAcquirerEnsureDownloadsVerifiesExtracts(t *testing.T) {
	archive := makeTarGz(t, "mailpit", []byte("#!/bin/sh\necho hi\n"))
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/mailpit.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	e := Engine{
		ID:      "mailpit",
		Version: "v1.30.3",
		Platforms: map[string]Platform{
			e_platformKey(): {
				URL:               srv.URL + "/mailpit.tar.gz",
				SHA256:            hexSum,
				ArchiveBinaryPath: "mailpit",
			},
		},
	}

	store := newTestDB(t)
	base := t.TempDir()
	a := NewAcquirer(store, base)

	path, err := a.Ensure(context.Background(), e)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if filepath.Dir(path) != filepath.Join(base, "mailpit", "v1.30.3") {
		t.Fatalf("unexpected path: %s", path)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Mode().Perm()&0o111 == 0 {
		t.Fatalf("binary not executable: %v", st.Mode())
	}

	if !a.IsInstalled(context.Background(), e) {
		t.Fatal("expected IsInstalled true after Ensure")
	}
}

func TestAcquirerRejectsBadChecksum(t *testing.T) {
	archive := makeTarGz(t, "mailpit", []byte("payload"))
	mux := http.NewServeMux()
	mux.HandleFunc("/mailpit.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(archive)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	e := Engine{
		ID:      "mailpit",
		Version: "v1.30.3",
		Platforms: map[string]Platform{
			e_platformKey(): {
				URL:               srv.URL + "/mailpit.tar.gz",
				SHA256:            "deadbeef",
				ArchiveBinaryPath: "mailpit",
			},
		},
	}
	a := NewAcquirer(newTestDB(t), t.TempDir())
	if _, err := a.Ensure(context.Background(), e); err == nil {
		t.Fatal("expected checksum mismatch error")
	}
}

func e_platformKey() string {
	return Engine{}.PlatformKey()
}
