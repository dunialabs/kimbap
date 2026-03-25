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
	Version       int       `yaml:"version"`
	LastSync      time.Time `yaml:"last_sync"`
	ArtifactHash  string    `yaml:"artifact_hash"`
	SyncedSkills  []string  `yaml:"synced_skills"`
}

// StaleCheckResult describes whether agent artifacts are out of date.
type StaleCheckResult struct {
	Stale        bool     `json:"stale"`
	NewSkills    []string `json:"new_skills,omitempty"`
	RemovedSkills []string `json:"removed_skills,omitempty"`
	LastSync     string   `json:"last_sync,omitempty"`
}

func syncStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(xdgConfigHome(home), "kimbap", "sync-state.yaml"), nil
}

// ReadSyncState loads the persisted sync state. Returns a zero-value state
// if the file does not exist.
func ReadSyncState() (*SyncState, error) {
	path, err := syncStatePath()
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
func WriteSyncState(state *SyncState) error {
	if state == nil {
		return fmt.Errorf("state is nil")
	}

	path, err := syncStatePath()
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

// RecordSync updates the sync state with current skill names and artifact hash.
// skillNames and artifactContents must be parallel slices of equal length.
func RecordSync(skillNames []string, artifactContents []string) error {
	sortedNames, sortedContents := sortByName(skillNames, artifactContents)
	state := &SyncState{
		Version:      1,
		LastSync:     time.Now().UTC(),
		ArtifactHash: computeArtifactHash(sortedContents),
		SyncedSkills: sortedNames,
	}
	return WriteSyncState(state)
}

// CheckStaleness compares current installed skills against the last recorded
// sync state. Returns whether artifacts are stale and what changed.
func CheckStaleness(currentSkillNames []string, currentArtifactContents []string) (*StaleCheckResult, error) {
	state, err := ReadSyncState()
	if err != nil {
		return nil, err
	}

	if state.LastSync.IsZero() {
		sorted := make([]string, len(currentSkillNames))
		copy(sorted, currentSkillNames)
		sort.Strings(sorted)
		return &StaleCheckResult{
			Stale:         true,
			NewSkills:     sorted,
			RemovedSkills: make([]string, 0),
		}, nil
	}

	currentSet := toStringSet(currentSkillNames)
	syncedSet := toStringSet(state.SyncedSkills)

	newSkills := make([]string, 0)
	removedSkills := make([]string, 0)
	for name := range currentSet {
		if !syncedSet[name] {
			newSkills = append(newSkills, name)
		}
	}
	for name := range syncedSet {
		if !currentSet[name] {
			removedSkills = append(removedSkills, name)
		}
	}

	sort.Strings(newSkills)
	sort.Strings(removedSkills)

	namesDiffer := len(newSkills) > 0 || len(removedSkills) > 0

	_, sortedContents := sortByName(currentSkillNames, currentArtifactContents)
	contentDiffers := state.ArtifactHash != computeArtifactHash(sortedContents)

	return &StaleCheckResult{
		Stale:         namesDiffer || contentDiffers,
		NewSkills:     newSkills,
		RemovedSkills: removedSkills,
		LastSync:      state.LastSync.Format(time.RFC3339),
	}, nil
}

// FormatStaleWarning returns a human-readable stderr warning.
// Returns empty string if not stale.
func FormatStaleWarning(result *StaleCheckResult) string {
	if result == nil || !result.Stale {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("warning: agent skills out of sync")

	changes := len(result.NewSkills) + len(result.RemovedSkills)
	if changes > 0 {
		sb.WriteString(fmt.Sprintf(" (%d change(s))", changes))
	}
	sb.WriteString(".\n")

	for _, name := range result.NewSkills {
		sb.WriteString(fmt.Sprintf("  + %s (new)\n", name))
	}
	for _, name := range result.RemovedSkills {
		sb.WriteString(fmt.Sprintf("  - %s (removed)\n", name))
	}

	sb.WriteString("  Run: kimbap agents setup   (global)\n")
	sb.WriteString("  Run: kimbap agents sync    (this project)\n")

	return sb.String()
}

func sortByName(names []string, contents []string) ([]string, []string) {
	type pair struct {
		name    string
		content string
	}
	pairs := make([]pair, len(names))
	for i, name := range names {
		content := ""
		if i < len(contents) {
			content = contents[i]
		}
		pairs[i] = pair{name: name, content: content}
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].name < pairs[j].name })
	sortedNames := make([]string, len(pairs))
	sortedContents := make([]string, len(pairs))
	for i, p := range pairs {
		sortedNames[i] = p.name
		sortedContents[i] = p.content
	}
	return sortedNames, sortedContents
}

func computeArtifactHash(contents []string) string {
	h := sha256.New()
	for _, c := range contents {
		h.Write([]byte(c))
		h.Write([]byte{0})
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil))
}

func toStringSet(items []string) map[string]bool {
	set := make(map[string]bool, len(items))
	for _, item := range items {
		set[item] = true
	}
	return set
}
