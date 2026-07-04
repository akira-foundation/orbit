package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

const (
	rootCommonName = "Orbit Local Development CA"
	leafCommonName = "*.orbit.test"
	caValidYears   = 10
	leafValidYears = 1
)

type Material struct {
	Dir        string
	CACertPath string
	CAKeyPath  string
	LeafCert   string
	LeafKey    string
}

func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "Orbit", "tls"), nil
}

// Ensure makes sure a CA + wildcard leaf cert exist on disk. Returns the
// resolved paths. Creates files when missing; regenerates the leaf when it's
// within 30 days of expiry so live dev never breaks on cert rotation.
func Ensure(domainSuffix string) (*Material, error) {
	dir, err := DefaultDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir tls dir: %w", err)
	}

	m := &Material{
		Dir:        dir,
		CACertPath: filepath.Join(dir, "orbit-ca.pem"),
		CAKeyPath:  filepath.Join(dir, "orbit-ca-key.pem"),
		LeafCert:   filepath.Join(dir, "orbit-leaf.pem"),
		LeafKey:    filepath.Join(dir, "orbit-leaf-key.pem"),
	}

	caCert, caKey, err := loadOrCreateCA(m)
	if err != nil {
		return nil, err
	}

	needLeaf := !exists(m.LeafCert) || !exists(m.LeafKey) ||
		leafExpiringSoon(m.LeafCert) || !leafHasChain(m.LeafCert)
	if needLeaf {
		if err := createLeaf(m, caCert, caKey, domainSuffix); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func loadOrCreateCA(m *Material) (*x509.Certificate, *rsa.PrivateKey, error) {
	if exists(m.CACertPath) && exists(m.CAKeyPath) {
		cert, err := readCert(m.CACertPath)
		if err == nil {
			key, kerr := readRSAKey(m.CAKeyPath)
			if kerr == nil {
				return cert, key, nil
			}
		}
		// Old ECDSA CA on disk — wipe and regen as RSA so macOS/Chrome accept it.
		_ = os.Remove(m.CACertPath)
		_ = os.Remove(m.CAKeyPath)
		_ = os.Remove(m.LeafCert)
		_ = os.Remove(m.LeafKey)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	serial, err := randomSerial()
	if err != nil {
		return nil, nil, err
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return nil, nil, err
	}
	ski := sha1.Sum(pubBytes)
	tmpl := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: rootCommonName, Organization: []string{"Orbit"}},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().AddDate(caValidYears, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
		SubjectKeyId:          ski[:],
		SignatureAlgorithm:    x509.SHA256WithRSA,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	if err := writePEM(m.CACertPath, "CERTIFICATE", der, 0o644); err != nil {
		return nil, nil, err
	}
	keyDER := x509.MarshalPKCS1PrivateKey(key)
	if err := writePEM(m.CAKeyPath, "RSA PRIVATE KEY", keyDER, 0o600); err != nil {
		return nil, nil, err
	}
	return cert, key, nil
}

func createLeaf(m *Material, caCert *x509.Certificate, caKey *rsa.PrivateKey, domainSuffix string) error {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	serial, err := randomSerial()
	if err != nil {
		return err
	}
	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return err
	}
	ski := sha1.Sum(pubBytes)
	wildcard := "*." + domainSuffix
	tmpl := &x509.Certificate{
		SerialNumber:       serial,
		Subject:            pkix.Name{CommonName: leafCommonName, Organization: []string{"Orbit"}},
		NotBefore:          time.Now().Add(-1 * time.Hour),
		NotAfter:           time.Now().AddDate(leafValidYears, 0, 0),
		KeyUsage:           x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:        []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		DNSNames:           []string{wildcard, domainSuffix, "localhost"},
		SubjectKeyId:       ski[:],
		AuthorityKeyId:     caCert.SubjectKeyId,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, caCert, &key.PublicKey, caKey)
	if err != nil {
		return err
	}
	if err := writeChain(m.LeafCert, der, caCert.Raw, 0o644); err != nil {
		return err
	}
	keyDER := x509.MarshalPKCS1PrivateKey(key)
	return writePEM(m.LeafKey, "RSA PRIVATE KEY", keyDER, 0o600)
}

func writeChain(path string, leafDER, caDER []byte, mode os.FileMode) error {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if err := pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: leafDER}); err != nil {
		return err
	}
	return pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: caDER})
}

func leafHasChain(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	count := 0
	for {
		block, rest := pem.Decode(b)
		if block == nil {
			break
		}
		if block.Type == "CERTIFICATE" {
			count++
		}
		b = rest
	}
	return count >= 2
}

func leafExpiringSoon(path string) bool {
	c, err := readCert(path)
	if err != nil {
		return true
	}
	return time.Until(c.NotAfter) < 30*24*time.Hour
}

func readCert(path string) (*x509.Certificate, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("no PEM block in " + path)
	}
	return x509.ParseCertificate(block.Bytes)
}

func readRSAKey(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, errors.New("no PEM block in " + path)
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func writePEM(path, blockType string, der []byte, mode os.FileMode) error {
	out, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	return pem.Encode(out, &pem.Block{Type: blockType, Bytes: der})
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func randomSerial() (*big.Int, error) {
	max := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, max)
}
