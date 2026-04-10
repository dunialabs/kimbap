package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/services"
)

type servicesActionRegistry struct {
	installer           *services.LocalInstaller
	verifyMode          string
	signaturePolicy     string
	servicesDir         string
	mu                  sync.RWMutex
	cachedDefs          []actions.ActionDefinition
	cachedByName        map[string]actions.ActionDefinition
	cacheFingerprint    string
	cachedManifestFiles []string
	lastFullScan        time.Time
	fullScanInterval    time.Duration
}

func (r *servicesActionRegistry) InvalidateActionDefinitionCache() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cachedDefs = nil
	r.cachedByName = nil
	r.cacheFingerprint = ""
	r.cachedManifestFiles = nil
	r.lastFullScan = time.Time{}
}

func (r *servicesActionRegistry) Lookup(_ context.Context, name string) (*actions.ActionDefinition, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("action name is required")
	}

	r.mu.RLock()
	if byName := r.cachedByName; byName != nil {
		if def, ok := byName[name]; ok {
			r.mu.RUnlock()
			copyDef := def
			return &copyDef, nil
		}
	}
	r.mu.RUnlock()

	if def, handled, err := r.lookupTargeted(name); err != nil || handled {
		return def, err
	}

	defs, err := r.loadDefinitions()
	if err != nil {
		return nil, err
	}

	for i := range defs {
		if defs[i].Name == name {
			return &defs[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %s", actions.ErrLookupNotFound, name)
}

func (r *servicesActionRegistry) lookupTargeted(name string) (*actions.ActionDefinition, bool, error) {
	serviceName, _, ok := strings.Cut(name, ".")
	if !ok {
		return nil, false, nil
	}

	installed, err := r.installer.Get(serviceName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, true, fmt.Errorf("%w: %s", actions.ErrLookupNotFound, name)
		}
		return nil, true, err
	}
	if !installed.Enabled {
		return nil, true, fmt.Errorf("%w: %s", actions.ErrLookupNotFound, name)
	}

	okToUse, verifyErr := r.verifyInstalledServiceWithWarnings(serviceName, true)
	if !okToUse {
		if verifyErr != nil {
			return nil, true, verifyErr
		}
		return nil, true, fmt.Errorf("%w: %s", actions.ErrLookupNotFound, name)
	}

	defs, convErr := services.ToActionDefinitions(&installed.Manifest)
	if convErr != nil {
		return nil, true, convErr
	}
	for i := range defs {
		if defs[i].Name == name {
			copyDef := defs[i]
			return &copyDef, true, nil
		}
	}
	return nil, true, fmt.Errorf("%w: %s", actions.ErrLookupNotFound, name)
}

func (r *servicesActionRegistry) List(_ context.Context, opts runtimepkg.ListOptions) ([]actions.ActionDefinition, error) {
	defs, err := r.loadDefinitions()
	if err != nil {
		return nil, err
	}

	namespace := strings.TrimSpace(opts.Namespace)
	resource := strings.TrimSpace(opts.Resource)
	verb := strings.TrimSpace(opts.Verb)
	filtered := make([]actions.ActionDefinition, 0, len(defs))
	for _, def := range defs {
		if namespace != "" && !strings.EqualFold(def.Namespace, namespace) {
			continue
		}
		if resource != "" && !strings.EqualFold(def.Resource, resource) {
			continue
		}
		if verb != "" && !strings.EqualFold(def.Verb, verb) {
			continue
		}
		filtered = append(filtered, def)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Name < filtered[j].Name
	})
	if opts.Limit > 0 && len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
	}
	return filtered, nil
}

func (r *servicesActionRegistry) computeFingerprint() (string, []string) {
	entries, err := os.ReadDir(r.servicesDir)
	if err != nil {
		return "", nil
	}
	var b strings.Builder
	manifestFiles := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		manifestFiles = append(manifestFiles, filepath.Join(r.servicesDir, e.Name()))
		fmt.Fprintf(&b, "%s:%d:%d;", e.Name(), info.Size(), info.ModTime().UnixNano())
	}
	lockPath := filepath.Join(r.servicesDir, "kimbap-services.lock")
	if info, err := os.Stat(lockPath); err == nil {
		fmt.Fprintf(&b, "lock:%d:%d", info.Size(), info.ModTime().UnixNano())
	}
	return b.String(), manifestFiles
}

func (r *servicesActionRegistry) loadDefinitions() ([]actions.ActionDefinition, error) {
	if r == nil || r.installer == nil {
		return nil, fmt.Errorf("services installer is not initialized")
	}
	scanInterval := r.fullScanInterval
	if scanInterval <= 0 {
		scanInterval = time.Second
	}
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cachedDefs != nil && now.Sub(r.lastFullScan) < scanInterval {
		return r.cachedDefs, nil
	}
	fp, manifestFiles := r.computeFingerprint()
	if fp != "" && fp == r.cacheFingerprint && r.cachedDefs != nil {
		r.cachedManifestFiles = manifestFiles
		r.lastFullScan = now
		return r.cachedDefs, nil
	}

	defs, err := r.loadDefinitionsUncached(true)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]actions.ActionDefinition, len(defs))
	for i := range defs {
		byName[defs[i].Name] = defs[i]
	}
	r.cachedDefs = defs
	r.cachedByName = byName
	r.cacheFingerprint = fp
	r.cachedManifestFiles = manifestFiles
	r.lastFullScan = now
	return defs, nil
}

func (r *servicesActionRegistry) loadDefinitionsUncached(emitWarnings bool) ([]actions.ActionDefinition, error) {
	installed, err := r.installer.ListEnabled()
	if err != nil {
		return nil, err
	}
	out := make([]actions.ActionDefinition, 0)
	for _, it := range installed {
		if ok, verifyErr := r.verifyInstalledServiceWithWarnings(it.Manifest.Name, emitWarnings); !ok {
			if verifyErr != nil {
				return nil, verifyErr
			}
			continue
		}
		defs, convErr := services.ToActionDefinitions(&it.Manifest)
		if convErr != nil {
			return nil, convErr
		}
		out = append(out, defs...)
	}
	return out, nil
}

func (r *servicesActionRegistry) verifyInstalledServiceWithWarnings(name string, emitWarnings bool) (bool, error) {
	verifyMode := normalizeVerifyMode(r.verifyMode)
	signaturePolicy := normalizeSignaturePolicy(r.signaturePolicy)

	if verifyMode == "off" && signaturePolicy != "required" {
		return true, nil
	}

	result, err := r.installer.Verify(name)
	if err != nil {
		if signaturePolicy == "required" {
			return false, fmt.Errorf("verify installed service %q for required signature policy: %w", name, err)
		}
		if verifyMode == "strict" {
			return false, fmt.Errorf("verify installed service %q: %w", name, err)
		}
		if emitWarnings {
			_, _ = fmt.Fprintf(os.Stderr, "warning: verify installed service %q failed: %v\n", name, err)
		}
		return true, nil
	}

	if signaturePolicy == "required" {
		if !result.Locked || !result.Signed || !result.SignatureValid {
			msg := fmt.Sprintf("service %q failed required signature verification (locked=%v signed=%v valid=%v)", name, result.Locked, result.Signed, result.SignatureValid)
			if verifyMode == "strict" {
				return false, fmt.Errorf("%s", msg)
			}
			if emitWarnings {
				_, _ = fmt.Fprintln(os.Stderr, "warning:", msg)
			}
			return false, nil
		}
	}

	if verifyMode != "off" {
		if !result.Locked || !result.Verified {
			msg := fmt.Sprintf("service %q failed digest verification (locked=%v verified=%v)", name, result.Locked, result.Verified)
			if verifyMode == "strict" {
				return false, fmt.Errorf("%s", msg)
			}
			if emitWarnings {
				_, _ = fmt.Fprintln(os.Stderr, "warning:", msg)
			}
			return true, nil
		}
	}

	return true, nil
}

func (r *servicesActionRegistry) commandExecutables() ([]string, error) {
	installed, err := r.installer.ListEnabled()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	out := make([]string, 0)
	for _, it := range installed {
		adapterType := strings.ToLower(strings.TrimSpace(it.Manifest.Adapter))
		if adapterType == "" {
			adapterType = "http"
		}
		if adapterType != "command" {
			continue
		}

		ok, verifyErr := r.verifyInstalledServiceWithWarnings(it.Manifest.Name, false)
		if !ok {
			if verifyErr != nil {
				return nil, verifyErr
			}
			continue
		}

		defs, convErr := services.ToActionDefinitions(&it.Manifest)
		if convErr != nil {
			return nil, convErr
		}
		for _, def := range defs {
			if strings.ToLower(strings.TrimSpace(def.Adapter.Type)) != "command" {
				continue
			}
			exe := strings.TrimSpace(def.Adapter.ExecutablePath)
			if exe == "" || seen[exe] {
				continue
			}
			seen[exe] = true
			out = append(out, exe)
		}
	}

	return out, nil
}

func normalizeVerifyMode(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case "off", "strict", "warn":
		return normalized
	default:
		return "warn"
	}
}

func normalizeSignaturePolicy(policy string) string {
	normalized := strings.ToLower(strings.TrimSpace(policy))
	switch normalized {
	case "off", "optional", "required":
		return normalized
	default:
		return "optional"
	}
}
