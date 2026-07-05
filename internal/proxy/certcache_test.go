package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeSelfSignedCert(t *testing.T, dir, name, cn string) (certPath, keyPath string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: cn},
		DNSNames:     []string{cn},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPath = filepath.Join(dir, name+".pem")
	keyPath = filepath.Join(dir, name+"-key.pem")
	certOut, err := os.Create(certPath)
	if err != nil {
		t.Fatal(err)
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatal(err)
	}
	keyOut, err := os.Create(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}); err != nil {
		t.Fatal(err)
	}
	return certPath, keyPath
}

func TestCertCacheReloadsWhenFileChanges(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := writeSelfSignedCert(t, dir, "leaf", "first.test")

	cache := newCertCache(certPath, keyPath)
	got, err := cache.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := x509.ParseCertificate(got.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if first.Subject.CommonName != "first.test" {
		t.Fatalf("CN = %q, want first.test", first.Subject.CommonName)
	}

	// simulates a leaf rotation while the proxy is already running
	time.Sleep(10 * time.Millisecond)
	writeSelfSignedCert(t, dir, "leaf", "second.test")

	got, err = cache.GetCertificate(nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := x509.ParseCertificate(got.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if second.Subject.CommonName != "second.test" {
		t.Fatalf("CN = %q, want second.test after reload", second.Subject.CommonName)
	}
}
