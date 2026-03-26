package services

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type InstalledService struct {
	Manifest    ServiceManifest
	InstalledAt time.Time
	Source      string
	Path        string
}

type LockEntry struct {
	Name      string    `yaml:"name"`
	Version   string    `yaml:"version"`
	Digest    string    `yaml:"digest"`
	Source    string    `yaml:"source"`
	Signature string    `yaml:"signature,omitempty"`
	LockedAt  time.Time `yaml:"locked_at"`
}

type Lockfile struct {
	Version   int                  `yaml:"version"`
	PublicKey string               `yaml:"public_key,omitempty"`
	Services  map[string]LockEntry `yaml:"services"`
}

type VerifyResult struct {
	Name           string `json:"name"`
	Verified       bool   `json:"verified"`
	ExpectedDigest string `json:"expected_digest,omitempty"`
	ActualDigest   string `json:"actual_digest,omitempty"`
	Locked         bool   `json:"locked"`
	SignatureValid bool   `json:"signature_valid"`
	Signed         bool   `json:"signed"`
}

type LocalInstaller struct {
	skillsDir string
}

func NewLocalInstaller(skillsDir string) *LocalInstaller {
	return &LocalInstaller{skillsDir: skillsDir}
}

var ErrServiceAlreadyInstalled = fmt.Errorf("service already installed")

func (i *LocalInstaller) Install(manifest *ServiceManifest, source string) (*InstalledService, error) {
	return i.InstallWithForce(manifest, source, false)
}

func (i *LocalInstaller) InstallWithForce(manifest *ServiceManifest, source string, force bool) (*InstalledService, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}
	if err := ValidateServiceName(manifest.Name); err != nil {
		return nil, err
	}
	if errs := ValidateManifest(manifest); len(errs) > 0 {
		return nil, validationErrorsToError("invalid manifest", errs)
	}
	if err := os.MkdirAll(i.skillsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create services dir: %w", err)
	}

	p := filepath.Join(i.skillsDir, manifest.Name+".yaml")
	if !force {
		if _, err := os.Stat(p); err == nil {
			return nil, ErrServiceAlreadyInstalled
		}
	}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest yaml: %w", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest file: %w", err)
	}
	digest := computeDigest(data)

	if strings.TrimSpace(source) == "" {
		source = "local"
	}

	lf, err := i.readLockfile()
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}
	lf.Services[manifest.Name] = LockEntry{
		Name:     manifest.Name,
		Version:  manifest.Version,
		Digest:   digest,
		Source:   source,
		LockedAt: time.Now().UTC(),
	}
	if err := i.writeLockfile(lf); err != nil {
		_ = os.Remove(p)
		return nil, fmt.Errorf("write lockfile: %w", err)
	}

	return &InstalledService{
		Manifest:    *manifest,
		InstalledAt: time.Now().UTC(),
		Source:      source,
		Path:        p,
	}, nil
}

func (i *LocalInstaller) Remove(name string) error {
	if i == nil {
		return fmt.Errorf("installer is nil")
	}
	if err := ValidateServiceName(name); err != nil {
		return err
	}

	lf, err := i.readLockfile()
	if err != nil {
		return fmt.Errorf("read lockfile: %w", err)
	}
	if _, ok := lf.Services[name]; ok {
		delete(lf.Services, name)
		if err := i.writeLockfile(lf); err != nil {
			return fmt.Errorf("write lockfile: %w", err)
		}
	}

	p := filepath.Join(i.skillsDir, name+".yaml")
	if err := os.Remove(p); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("remove manifest file: %w", err)
		}
	}
	return nil
}

func (i *LocalInstaller) List() ([]InstalledService, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	entries, err := os.ReadDir(i.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledService{}, nil
		}
		return nil, fmt.Errorf("read services dir: %w", err)
	}

	out := make([]InstalledService, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		installed, err := i.Get(name)
		if err != nil {
			return nil, err
		}
		out = append(out, *installed)
	}

	sort.Slice(out, func(a, b int) bool {
		return out[a].Manifest.Name < out[b].Manifest.Name
	})

	return out, nil
}

func ValidateServiceName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("service name is required")
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid service name %q: must not contain path separators or '..'", name)
	}
	return nil
}

func (i *LocalInstaller) Get(name string) (*InstalledService, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	if err := ValidateServiceName(name); err != nil {
		return nil, err
	}
	p := filepath.Join(i.skillsDir, name+".yaml")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read installed service manifest: %w", err)
	}

	manifest, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("parse installed service manifest: %w", err)
	}

	fi, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("stat installed service manifest: %w", err)
	}

	return &InstalledService{
		Manifest:    *manifest,
		InstalledAt: fi.ModTime().UTC(),
		Source:      "local",
		Path:        p,
	}, nil
}

func (i *LocalInstaller) Verify(name string) (*VerifyResult, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	if err := ValidateServiceName(name); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(i.skillsDir, name+".yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read installed service manifest: %w", err)
	}
	actualDigest := computeDigest(data)

	lf, err := i.readLockfile()
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	entry, ok := lf.Services[name]
	if !ok {
		return &VerifyResult{
			Name:           name,
			Verified:       false,
			ActualDigest:   actualDigest,
			Locked:         false,
			SignatureValid: false,
			Signed:         false,
		}, nil
	}

	verified := strings.TrimSpace(entry.Digest) != "" && entry.Digest == actualDigest
	result := &VerifyResult{
		Name:           name,
		Verified:       verified,
		ExpectedDigest: entry.Digest,
		ActualDigest:   actualDigest,
		Locked:         true,
	}

	result.Signed = strings.TrimSpace(entry.Signature) != ""
	if result.Signed && strings.TrimSpace(lf.PublicKey) != "" {
		pubKeyBytes, decErr := hex.DecodeString(lf.PublicKey)
		if decErr == nil && len(pubKeyBytes) == ed25519.PublicKeySize {
			result.SignatureValid = verifySignature(ed25519.PublicKey(pubKeyBytes), entry.Digest, entry.Signature)
		}
	}

	return result, nil
}

func (i *LocalInstaller) VerifyWithKey(name string, pinnedPubKey ed25519.PublicKey) (*VerifyResult, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	if err := ValidateServiceName(name); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(i.skillsDir, name+".yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read installed service manifest: %w", err)
	}
	actualDigest := computeDigest(data)

	lf, err := i.readLockfile()
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	entry, ok := lf.Services[name]
	if !ok {
		return &VerifyResult{
			Name:         name,
			ActualDigest: actualDigest,
		}, nil
	}

	verified := strings.TrimSpace(entry.Digest) != "" && entry.Digest == actualDigest
	result := &VerifyResult{
		Name:           name,
		Verified:       verified,
		ExpectedDigest: entry.Digest,
		ActualDigest:   actualDigest,
		Locked:         true,
		Signed:         strings.TrimSpace(entry.Signature) != "",
	}

	if result.Signed {
		result.SignatureValid = verifySignature(pinnedPubKey, entry.Digest, entry.Signature)
	}

	return result, nil
}

func (i *LocalInstaller) Sign(privateKey ed25519.PrivateKey) error {
	lf, err := i.readLockfile()
	if err != nil {
		return err
	}

	pubKey := privateKey.Public().(ed25519.PublicKey)
	lf.PublicKey = hex.EncodeToString(pubKey)

	for name, entry := range lf.Services {
		if strings.TrimSpace(entry.Digest) == "" {
			continue
		}
		entry.Signature = signDigest(privateKey, entry.Digest)
		lf.Services[name] = entry
	}

	return i.writeLockfile(lf)
}

func (i *LocalInstaller) lockfilePath() string {
	return filepath.Join(i.skillsDir, "kimbap-services.lock")
}

func (i *LocalInstaller) readLockfile() (*Lockfile, error) {
	data, err := os.ReadFile(i.lockfilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Lockfile{Version: 1, Services: map[string]LockEntry{}}, nil
		}
		return nil, err
	}

	var lf Lockfile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parse lockfile: %w", err)
	}
	if lf.Version == 0 {
		lf.Version = 1
	}
	if lf.Services == nil {
		lf.Services = map[string]LockEntry{}
	}
	return &lf, nil
}

func (i *LocalInstaller) writeLockfile(lf *Lockfile) error {
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	return os.WriteFile(i.lockfilePath(), data, 0o644)
}

func computeDigest(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func signDigest(privateKey ed25519.PrivateKey, digest string) string {
	sig := ed25519.Sign(privateKey, []byte(digest))
	return hex.EncodeToString(sig)
}

func verifySignature(publicKey ed25519.PublicKey, digest, signature string) bool {
	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	return ed25519.Verify(publicKey, []byte(digest), sigBytes)
}
