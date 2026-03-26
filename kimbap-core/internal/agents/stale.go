package agents

import (
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

// SyncState tracks when the last global/project sync was performed
// and a hash of the rendered artifacts at that time.
type SyncState struct {
	Version        int       `yaml:"version"`
	LastSync       time.Time `yaml:"last_sync"`
	ArtifactHash   string    `yaml:"artifact_hash"`
	SyncedServices []string  `yaml:"synced_skills"`
}

// StaleCheckResult describes whether agent artifacts are out of date.
type StaleCheckResult struct {
	Stale           bool     `json:"stale"`
	NewServices     []string `json:"new_services,omitempty"`
	RemovedServices []string `json:"removed_services,omitempty"`
	LastSync        string   `json:"last_sync,omitempty"`
}

func syncStatePath(scope string) (string, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "global"
	}
	sum := sha256.Sum256([]byte(scope))
	fileName := hex.EncodeToString(sum[:]) + ".yaml"
	if env := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); env != "" {
		return filepath.Join(env, "kimbap", "sync-state", fileName), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "kimbap", "sync-state", fileName), nil
}

// ReadSyncState loads the persisted sync state. Returns a zero-value state
// if the file does not exist.
func ReadSyncState(scope string) (*SyncState, error) {
	path, err := syncStatePath(scope)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &SyncState{Version: 1}, nil
		}
		return nil, fmt.Errorf("read sync state: %w", err)
	}

	var state SyncState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse sync state: %w", err)
	}
	if state.Version == 0 {
		state.Version = 1
	}
	return &state, nil
}

// WriteSyncState persists the sync state atomically.
func WriteSyncState(scope string, state *SyncState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	path, err := syncStatePath(scope)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create sync state directory: %w", err)
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal sync state: %w", err)
	}
	return atomicWriteFile(path, string(data))
}

// RecordSync updates the sync state with current service names and artifact hash.
// serviceNames and artifactContents must be parallel slices of equal length.
func RecordSync(scope string, serviceNames []string, artifactContents []string) error {
	sortedNames, sortedContents, err := sortByName(serviceNames, artifactContents)
	if err != nil {
		return err
	}
	state := &SyncState{
		Version:        1,
		LastSync:       time.Now().UTC(),
		ArtifactHash:   computeArtifactHash(sortedContents),
		SyncedServices: sortedNames,
	}
	return WriteSyncState(scope, state)
}

// CheckStaleness compares current installed services against the last recorded
// sync state. Returns whether artifacts are stale and what changed.
func CheckStaleness(scope string, currentServiceNames []string, currentArtifactContents []string) (*StaleCheckResult, error) {
	state, err := ReadSyncState(scope)
	if err != nil {
		return nil, err
	}

	if state.LastSync.IsZero() {
		sorted := make([]string, len(currentServiceNames))
		copy(sorted, currentServiceNames)
		sort.Strings(sorted)
		return &StaleCheckResult{
			Stale:           true,
			NewServices:     sorted,
			RemovedServices: make([]string, 0),
		}, nil
	}

	currentSet := toStringSet(currentServiceNames)
	syncedSet := toStringSet(state.SyncedServices)

	newServices := make([]string, 0)
	removedServices := make([]string, 0)
	for name := range currentSet {
		if !syncedSet[name] {
			newServices = append(newServices, name)
		}
	}
	for name := range syncedSet {
		if !currentSet[name] {
			removedServices = append(removedServices, name)
		}
	}

	sort.Strings(newServices)
	sort.Strings(removedServices)

	namesDiffer := len(newServices) > 0 || len(removedServices) > 0

	_, sortedContents, sortErr := sortByName(currentServiceNames, currentArtifactContents)
	if sortErr != nil {
		return nil, sortErr
	}
	contentDiffers := state.ArtifactHash != computeArtifactHash(sortedContents)

	return &StaleCheckResult{
		Stale:           namesDiffer || contentDiffers,
		NewServices:     newServices,
		RemovedServices: removedServices,
		LastSync:        state.LastSync.Format(time.RFC3339),
	}, nil
}

// FormatStaleWarning returns a human-readable stderr warning.
// Returns empty string if not stale.
func FormatStaleWarning(result *StaleCheckResult) string {
	if result == nil || !result.Stale {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("warning: agent services out of sync")

	changes := len(result.NewServices) + len(result.RemovedServices)
	switch {
	case changes > 0:
		_, _ = fmt.Fprintf(&sb, " (%d change(s))", changes)
	default:
		sb.WriteString(" (service content changed)")
	}
	sb.WriteString(".\n")

	for _, name := range result.NewServices {
		_, _ = fmt.Fprintf(&sb, "  + %s (new)\n", name)
	}
	for _, name := range result.RemovedServices {
		_, _ = fmt.Fprintf(&sb, "  - %s (removed)\n", name)
	}
	_, _ = fmt.Fprintf(&sb, "  Run: kimbap agents sync    (this project)\n")

	return sb.String()
}

func sortByName(names []string, contents []string) ([]string, []string, error) {
	if len(names) != len(contents) {
		return nil, nil, fmt.Errorf("sortByName: names (%d) and contents (%d) must have equal length", len(names), len(contents))
	}
	type pair struct {
		name    string
		content string
	}
	pairs := make([]pair, len(names))
	for i, name := range names {
		pairs[i] = pair{name: name, content: contents[i]}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].name < pairs[j].name })
	sortedNames := make([]string, len(pairs))
	sortedContents := make([]string, len(pairs))
	for i, p := range pairs {
		sortedNames[i] = p.name
		sortedContents[i] = p.content
	}
	return sortedNames, sortedContents, nil
}

func computeArtifactHash(contents []string) string {
	h := sha256.New()
	for _, c := range contents {
		h.Write([]byte(c))
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func computePackArtifactHash(packs []InstalledServicePack) string {
	h := sha256.New()
	sorted := make([]InstalledServicePack, len(packs))
	copy(sorted, packs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, pack := range sorted {
		h.Write([]byte(pack.Name))
		h.Write([]byte{0})
		fileNames := make([]string, 0, len(pack.PackFiles)+1)
		if pack.SkillMD != "" {
			fileNames = append(fileNames, "SKILL.md")
		} else if _, ok := pack.PackFiles["SKILL.md"]; ok {
			fileNames = append(fileNames, "SKILL.md")
		}
		for name := range pack.PackFiles {
			if name != "SKILL.md" {
				fileNames = append(fileNames, name)
			}
		}
		sort.Strings(fileNames)
		for _, name := range fileNames {
			h.Write([]byte(name))
			h.Write([]byte{0})
			var content string
			if name == "SKILL.md" {
				if pack.SkillMD != "" {
					content = pack.SkillMD
				} else {
					content = pack.PackFiles[name]
				}
			} else {
				content = pack.PackFiles[name]
			}
			h.Write([]byte(content))
			h.Write([]byte{0})
		}
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func RecordSyncPacks(scope string, packs []InstalledServicePack) error {
	names := make([]string, len(packs))
	for i, p := range packs {
		names[i] = p.Name
	}
	sort.Strings(names)
	state := &SyncState{
		Version:        1,
		LastSync:       time.Now().UTC(),
		ArtifactHash:   computePackArtifactHash(packs),
		SyncedServices: names,
	}
	return WriteSyncState(scope, state)
}

func toStringSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}
