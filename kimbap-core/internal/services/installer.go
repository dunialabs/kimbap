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
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type InstalledService struct {
	Manifest    ServiceManifest
	InstalledAt time.Time
	Source      string
	Enabled     bool
	Path        string
}

type LockEntry struct {
	Name      string    `yaml:"name"`
	Version   string    `yaml:"version"`
	Digest    string    `yaml:"digest"`
	Source    string    `yaml:"source"`
	Signature string    `yaml:"signature,omitempty"`
	Enabled   bool      `yaml:"enabled"`
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
	mu        sync.RWMutex
}

func NewLocalInstaller(skillsDir string) *LocalInstaller {
	return &LocalInstaller{skillsDir: skillsDir}
}

var ErrServiceAlreadyInstalled = fmt.Errorf("service already installed")

func (i *LocalInstaller) Install(manifest *ServiceManifest, source string) (*InstalledService, error) {
	return i.InstallWithForce(manifest, source, false)
}

func (i *LocalInstaller) InstallWithForce(manifest *ServiceManifest, source string, force bool) (*InstalledService, error) {
	return i.InstallWithForceAndActivation(manifest, source, force, true)
}

func (i *LocalInstaller) InstallWithForceAndActivation(manifest *ServiceManifest, source string, force bool, enabled bool) (*InstalledService, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
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
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat manifest file: %w", err)
		}
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest yaml: %w", err)
	}
	digest := computeDigest(data)

	if strings.TrimSpace(source) == "" {
		source = "local"
	}

	existingData, hadExisting, err := readInstalledManifestFile(p)
	if err != nil {
		return nil, err
	}

	lf, err := i.readLockfile()
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}
	if err := writeInstalledManifestFile(p, data); err != nil {
		return nil, fmt.Errorf("write manifest file: %w", err)
	}

	lf.Services[manifest.Name] = LockEntry{
		Name:     manifest.Name,
		Version:  manifest.Version,
		Digest:   digest,
		Source:   source,
		Enabled:  enabled,
		LockedAt: time.Now().UTC(),
	}
	if err := i.writeLockfile(lf); err != nil {
		if restoreErr := restoreInstalledManifestFile(p, existingData, hadExisting); restoreErr != nil {
			return nil, fmt.Errorf("write lockfile: %w (restore manifest: %v)", err, restoreErr)
		}
		return nil, fmt.Errorf("write lockfile: %w", err)
	}

	return &InstalledService{
		Manifest:    *manifest,
		InstalledAt: time.Now().UTC(),
		Source:      source,
		Enabled:     enabled,
		Path:        p,
	}, nil
}

func (i *LocalInstaller) Enable(name string) error {
	return i.setEnabled(name, true)
}

func (i *LocalInstaller) Disable(name string) error {
	return i.setEnabled(name, false)
}

func (i *LocalInstaller) setEnabled(name string, enabled bool) error {
	if i == nil {
		return fmt.Errorf("installer is nil")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if err := ValidateServiceName(name); err != nil {
		return err
	}

	lf, err := i.readLockfile()
	if err != nil {
		return fmt.Errorf("read lockfile: %w", err)
	}
	entry, ok := lf.Services[name]
	if !ok {
		return fmt.Errorf("service %q is not installed. Run 'kimbap service list' to see installed services", name)
	}
	entry.Enabled = enabled
	lf.Services[name] = entry
	if err := i.writeLockfile(lf); err != nil {
		return fmt.Errorf("write lockfile: %w", err)
	}
	return nil
}

func (i *LocalInstaller) Remove(name string) error {
	if i == nil {
		return fmt.Errorf("installer is nil")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	if err := ValidateServiceName(name); err != nil {
		return err
	}

	lf, err := i.readLockfile()
	if err != nil {
		return fmt.Errorf("read lockfile: %w", err)
	}
	_, hasLockEntry := lf.Services[name]

	p := filepath.Join(i.skillsDir, name+".yaml")
	existingData, hadManifest, err := readInstalledManifestFile(p)
	if err != nil {
		return err
	}
	if hadManifest {
		if err := os.Remove(p); err != nil {
			return fmt.Errorf("remove manifest file: %w", err)
		}
	}

	if hasLockEntry {
		delete(lf.Services, name)
		if err := i.writeLockfile(lf); err != nil {
			if restoreErr := restoreInstalledManifestFile(p, existingData, hadManifest); restoreErr != nil {
				return fmt.Errorf("write lockfile: %w (restore manifest: %v)", err, restoreErr)
			}
			return fmt.Errorf("write lockfile: %w", err)
		}
	}
	return nil
}

func (i *LocalInstaller) List() ([]InstalledService, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
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
		installed, err := i.getNoLock(name)
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

func (i *LocalInstaller) ListEnabled() ([]InstalledService, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}

	installed, err := i.List()
	if err != nil {
		return nil, err
	}

	enabledOnly := make([]InstalledService, 0, len(installed))
	for _, svc := range installed {
		if svc.Enabled {
			enabledOnly = append(enabledOnly, svc)
		}
	}

	return enabledOnly, nil
}

func ValidateServiceName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("service name is required")
	}
	if name == "kimbap" {
		return fmt.Errorf("service name %q is reserved", name)
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
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.getNoLock(name)
}

func (i *LocalInstaller) getNoLock(name string) (*InstalledService, error) {
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

	source := "local"
	enabled := true
	lf, err := i.readLockfile()
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}
	if entry, ok := lf.Services[name]; ok {
		if strings.TrimSpace(entry.Source) != "" {
			source = entry.Source
		}
		enabled = entry.Enabled
	}

	return &InstalledService{
		Manifest:    *manifest,
		InstalledAt: fi.ModTime().UTC(),
		Source:      source,
		Enabled:     enabled,
		Path:        p,
	}, nil
}

func (i *LocalInstaller) Verify(name string) (*VerifyResult, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	_, entry, result, err := i.verifyBuildContext(name)
	if err != nil {
		return nil, err
	}

	if !result.Locked {
		return &result, nil
	}

	result.Verified = strings.TrimSpace(entry.Digest) != "" && entry.Digest == result.ActualDigest
	if result.Signed {
		lf, readErr := i.readLockfile()
		if readErr != nil {
			return nil, fmt.Errorf("read lockfile: %w", readErr)
		}
		if strings.TrimSpace(lf.PublicKey) == "" {
			return &result, nil
		}
		pubKeyBytes, decErr := hex.DecodeString(lf.PublicKey)
		if decErr != nil {
			result.SignatureValid = false
			return &result, fmt.Errorf("decode embedded public key: %w", decErr)
		}
		sigValid, sigErr := verifySignature(ed25519.PublicKey(pubKeyBytes), entry.Digest, entry.Signature)
		if sigErr != nil {
			result.SignatureValid = false
			return &result, fmt.Errorf("verify signature: %w", sigErr)
		}
		result.SignatureValid = sigValid
	}

	return &result, nil
}

func (i *LocalInstaller) VerifyWithKey(name string, pinnedPubKey ed25519.PublicKey) (*VerifyResult, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	_, entry, result, err := i.verifyBuildContext(name)
	if err != nil {
		return nil, err
	}
	if !result.Locked {
		return &result, nil
	}

	result.Verified = strings.TrimSpace(entry.Digest) != "" && entry.Digest == result.ActualDigest

	if result.Signed {
		sigValid, sigErr := verifySignature(pinnedPubKey, entry.Digest, entry.Signature)
		if sigErr != nil {
			result.SignatureValid = false
			return &result, fmt.Errorf("verify signature: %w", sigErr)
		}
		result.SignatureValid = sigValid
	}

	return &result, nil
}

func (i *LocalInstaller) verifyBuildContext(name string) (digest string, entry LockEntry, result VerifyResult, err error) {
	if i == nil {
		err = fmt.Errorf("installer is nil")
		return
	}
	if validateErr := ValidateServiceName(name); validateErr != nil {
		err = validateErr
		return
	}

	manifestPath := filepath.Join(i.skillsDir, name+".yaml")
	data, readErr := os.ReadFile(manifestPath)
	if readErr != nil {
		err = fmt.Errorf("read installed service manifest: %w", readErr)
		return
	}
	digest = computeDigest(data)

	lf, lockErr := i.readLockfile()
	if lockErr != nil {
		err = fmt.Errorf("read lockfile: %w", lockErr)
		return
	}

	result = VerifyResult{
		Name:         name,
		ActualDigest: digest,
	}

	entry, result.Locked = lf.Services[name]
	if result.Locked {
		result.ExpectedDigest = entry.Digest
		result.Signed = strings.TrimSpace(entry.Signature) != ""
	}

	return
}

func (i *LocalInstaller) Sign(privateKey ed25519.PrivateKey) error {
	if i == nil {
		return fmt.Errorf("installer is nil")
	}
	i.mu.Lock()
	defer i.mu.Unlock()
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

	var disk lockfileDisk
	if err := yaml.Unmarshal(data, &disk); err != nil {
		return nil, fmt.Errorf("parse lockfile: %w", err)
	}

	lf := Lockfile{
		Version:   disk.Version,
		PublicKey: disk.PublicKey,
		Services:  make(map[string]LockEntry, len(disk.Services)),
	}
	for name, entry := range disk.Services {
		enabled := true
		if entry.Enabled != nil {
			enabled = *entry.Enabled
		}
		lf.Services[name] = LockEntry{
			Name:      entry.Name,
			Version:   entry.Version,
			Digest:    entry.Digest,
			Source:    entry.Source,
			Signature: entry.Signature,
			Enabled:   enabled,
			LockedAt:  entry.LockedAt,
		}
	}
	if lf.Version == 0 {
		lf.Version = 1
	}
	if lf.Services == nil {
		lf.Services = map[string]LockEntry{}
	}
	return &lf, nil
}

type lockEntryDisk struct {
	Name      string    `yaml:"name"`
	Version   string    `yaml:"version"`
	Digest    string    `yaml:"digest"`
	Source    string    `yaml:"source"`
	Signature string    `yaml:"signature,omitempty"`
	Enabled   *bool     `yaml:"enabled,omitempty"`
	LockedAt  time.Time `yaml:"locked_at"`
}

type lockfileDisk struct {
	Version   int                      `yaml:"version"`
	PublicKey string                   `yaml:"public_key,omitempty"`
	Services  map[string]lockEntryDisk `yaml:"services"`
}

func (i *LocalInstaller) writeLockfile(lf *Lockfile) error {
	data, err := yaml.Marshal(lf)
	if err != nil {
		return fmt.Errorf("marshal lockfile: %w", err)
	}
	lockPath := i.lockfilePath()
	if _, err := os.Stat(lockPath); err == nil {
		f, openErr := os.OpenFile(lockPath, os.O_WRONLY, 0)
		if openErr != nil {
			return fmt.Errorf("open lockfile for write: %w", openErr)
		}
		_ = f.Close()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat lockfile: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(lockPath), ".kimbap-services-lock-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp lockfile: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp lockfile: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp lockfile: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp lockfile: %w", err)
	}
	if err := os.Rename(tmpPath, lockPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp lockfile: %w", err)
	}
	return nil
}

func readInstalledManifestFile(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true, nil
	}
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return nil, false, fmt.Errorf("read manifest file: %w", err)
}

func restoreInstalledManifestFile(path string, data []byte, hadFile bool) error {
	if !hadFile {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove manifest file: %w", err)
		}
		return nil
	}
	if err := writeInstalledManifestFile(path, data); err != nil {
		return fmt.Errorf("write manifest file: %w", err)
	}
	return nil
}

func writeInstalledManifestFile(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".kimbap-service-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp manifest file: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp manifest file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp manifest file: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp manifest file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename temp manifest file: %w", err)
	}
	return nil
}

func computeDigest(data []byte) string {
	h := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(h[:])
}

func signDigest(privateKey ed25519.PrivateKey, digest string) string {
	sig := ed25519.Sign(privateKey, []byte(digest))
	return hex.EncodeToString(sig)
}

func verifySignature(publicKey ed25519.PublicKey, digest, signature string) (bool, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key length: expected %d, got %d", ed25519.PublicKeySize, len(publicKey))
	}

	sigBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, nil
	}

	return ed25519.Verify(publicKey, []byte(digest), sigBytes), nil
}
