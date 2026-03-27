package proxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type CAConfig struct {
	CertPEM  []byte
	KeyPEM   []byte
	CertPath string
	KeyPath  string
}

func GenerateCA(dataDir string) (*CAConfig, error) {
	certPath := filepath.Join(dataDir, "ca.crt")
	keyPath := filepath.Join(dataDir, "ca.key")
	if fileExists(certPath) && fileExists(keyPath) {
		return LoadCA(dataDir)
	}

	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate rsa key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("generate serial: %w", err)
	}

	now := time.Now().UTC()
	tmpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "Kimbap Local MITM CA",
			Organization: []string{"Kimbap"},
		},
		NotBefore:             now.Add(-1 * time.Hour),
		NotAfter:              now.AddDate(10, 0, 0),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("create certificate: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	if err := os.WriteFile(certPath, certPEM, 0o644); err != nil {
		return nil, fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(keyPath, keyPEM, 0o600); err != nil {
		return nil, fmt.Errorf("write key: %w", err)
	}

	return &CAConfig{CertPEM: certPEM, KeyPEM: keyPEM, CertPath: certPath, KeyPath: keyPath}, nil
}

func LoadCA(dataDir string) (*CAConfig, error) {
	certPath := filepath.Join(dataDir, "ca.crt")
	keyPath := filepath.Join(dataDir, "ca.key")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("read cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read key: %w", err)
	}

	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		return nil, fmt.Errorf("CA cert and key do not match: %w", err)
	}

	return &CAConfig{CertPEM: certPEM, KeyPEM: keyPEM, CertPath: certPath, KeyPath: keyPath}, nil
}

func TrustInstructions(certPath string) string {
	switch runtime.GOOS {
	case "darwin":
		return fmt.Sprintf("sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain %q", certPath)
	case "linux":
		return linuxTrustInstructions(certPath)
	case "windows":
		return fmt.Sprintf("certutil -addstore -f Root %q", certPath)
	default:
		return fmt.Sprintf("Trust CA certificate manually: %q", certPath)
	}
}

func linuxTrustInstructions(certPath string) string {
	type trustMethod struct {
		probe   string
		command string
	}
	methods := []trustMethod{
		{"update-ca-certificates", fmt.Sprintf("sudo cp %q /usr/local/share/ca-certificates/kimbap-ca.crt && sudo update-ca-certificates", certPath)},
		{"update-ca-trust", fmt.Sprintf("sudo cp %q /etc/pki/ca-trust/source/anchors/kimbap-ca.crt && sudo update-ca-trust extract", certPath)},
		{"trust", fmt.Sprintf("sudo trust anchor %q", certPath)},
	}
	for _, m := range methods {
		if hasCommand(m.probe) {
			return m.command
		}
	}
	return fmt.Sprintf("Copy CA certificate to your system trust store and update trust:\n  %q\n"+
		"  Debian/Ubuntu: sudo cp <cert> /usr/local/share/ca-certificates/ && sudo update-ca-certificates\n"+
		"  RHEL/Fedora:   sudo cp <cert> /etc/pki/ca-trust/source/anchors/ && sudo update-ca-trust extract\n"+
		"  Arch:          sudo trust anchor <cert>", certPath)
}

func hasCommand(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parseCACertificate(certPEM []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("invalid CA certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}
	if !cert.IsCA {
		return nil, errors.New("certificate is not a CA")
	}
	return cert, nil
}

func parseCAPrivateKey(keyPEM []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, errors.New("invalid CA key PEM")
	}
	if block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("unsupported key type %q", block.Type)
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse CA key: %w", err)
	}
	return key, nil
}
