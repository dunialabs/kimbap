package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type InstalledSkill struct {
	Manifest    SkillManifest
	InstalledAt time.Time
	Source      string
	Path        string
}

type LocalInstaller struct {
	skillsDir string
}

func NewLocalInstaller(skillsDir string) *LocalInstaller {
	return &LocalInstaller{skillsDir: skillsDir}
}

var ErrSkillAlreadyInstalled = fmt.Errorf("skill already installed")

func (i *LocalInstaller) Install(manifest *SkillManifest, source string) (*InstalledSkill, error) {
	return i.InstallWithForce(manifest, source, false)
}

func (i *LocalInstaller) InstallWithForce(manifest *SkillManifest, source string, force bool) (*InstalledSkill, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}
	if err := validateSkillName(manifest.Name); err != nil {
		return nil, err
	}
	if errs := ValidateManifest(manifest); len(errs) > 0 {
		return nil, validationErrorsToError("invalid manifest", errs)
	}
	if err := os.MkdirAll(i.skillsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create skills dir: %w", err)
	}

	p := filepath.Join(i.skillsDir, manifest.Name+".yaml")
	if !force {
		if _, err := os.Stat(p); err == nil {
			return nil, ErrSkillAlreadyInstalled
		}
	}
	data, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest yaml: %w", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return nil, fmt.Errorf("write manifest file: %w", err)
	}

	if strings.TrimSpace(source) == "" {
		source = "local"
	}

	return &InstalledSkill{
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
	if err := validateSkillName(name); err != nil {
		return err
	}
	p := filepath.Join(i.skillsDir, name+".yaml")
	if err := os.Remove(p); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("remove manifest file: %w", err)
	}
	return nil
}

func (i *LocalInstaller) List() ([]InstalledSkill, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	entries, err := os.ReadDir(i.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledSkill{}, nil
		}
		return nil, fmt.Errorf("read skills dir: %w", err)
	}

	out := make([]InstalledSkill, 0)
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

func validateSkillName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("skill name is required")
	}
	if strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
		return fmt.Errorf("invalid skill name %q: must not contain path separators or '..'", name)
	}
	return nil
}

func (i *LocalInstaller) Get(name string) (*InstalledSkill, error) {
	if i == nil {
		return nil, fmt.Errorf("installer is nil")
	}
	if err := validateSkillName(name); err != nil {
		return nil, err
	}
	p := filepath.Join(i.skillsDir, name+".yaml")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read installed skill manifest: %w", err)
	}

	manifest, err := ParseManifest(data)
	if err != nil {
		return nil, fmt.Errorf("parse installed skill manifest: %w", err)
	}

	fi, err := os.Stat(p)
	if err != nil {
		return nil, fmt.Errorf("stat installed skill manifest: %w", err)
	}

	return &InstalledSkill{
		Manifest:    *manifest,
		InstalledAt: fi.ModTime().UTC(),
		Source:      "local",
		Path:        p,
	}, nil
}
