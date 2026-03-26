package agents

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestComputeArtifactHash(t *testing.T) {
	tests := []struct {
		name      string
		contents  []string
		again     []string
		different []string
	}{
		{
			name:      "deterministic and order-sensitive",
			contents:  []string{"a", "b", "c"},
			again:     []string{"a", "b", "c"},
			different: []string{"c", "b", "a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := computeArtifactHash(tt.contents)
			second := computeArtifactHash(tt.again)
			if first != second {
				t.Fatalf("expected deterministic hash, got %q and %q", first, second)
			}

			third := computeArtifactHash(tt.different)
			if first == third {
				t.Fatalf("expected order-sensitive hash, got same hash %q", first)
			}

			if !strings.HasPrefix(first, "sha256:") {
				t.Fatalf("expected sha256 prefix, got %q", first)
			}
		})
	}
}

func TestSyncStateReadWrite(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)

	state, err := ReadSyncState("project-a")
	if err != nil {
		t.Fatalf("read initial sync state: %v", err)
	}
	if state.Version != 1 {
		t.Fatalf("expected default version=1, got %d", state.Version)
	}
	if !state.LastSync.IsZero() {
		t.Fatalf("expected zero LastSync for missing file, got %s", state.LastSync)
	}

	now := time.Now().UTC().Truncate(time.Second)
	write := &SyncState{
		Version:        1,
		LastSync:       now,
		ArtifactHash:   "sha256:abc",
		SyncedServices: []string{"github", "slack"},
	}
	if err := WriteSyncState("project-a", write); err != nil {
		t.Fatalf("write sync state: %v", err)
	}

	readBack, err := ReadSyncState("project-a")
	if err != nil {
		t.Fatalf("read sync state after write: %v", err)
	}
	if readBack.Version != 1 {
		t.Fatalf("expected version=1, got %d", readBack.Version)
	}
	if !readBack.LastSync.Equal(now) {
		t.Fatalf("expected LastSync=%s, got %s", now, readBack.LastSync)
	}
	if readBack.ArtifactHash != "sha256:abc" {
		t.Fatalf("unexpected artifact hash: %q", readBack.ArtifactHash)
	}
	if !reflect.DeepEqual(readBack.SyncedServices, []string{"github", "slack"}) {
		t.Fatalf("unexpected synced services: %+v", readBack.SyncedServices)
	}
}

func TestSyncStateScopesDoNotCollide(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("HOME", tmp)

	if err := RecordSync("project-a", []string{"github"}, []string{"a"}); err != nil {
		t.Fatalf("record project-a sync: %v", err)
	}
	if err := RecordSync("project-b", []string{"slack"}, []string{"b"}); err != nil {
		t.Fatalf("record project-b sync: %v", err)
	}

	result, err := CheckStaleness("project-a", []string{"github"}, []string{"a"})
	if err != nil {
		t.Fatalf("check project-a staleness: %v", err)
	}
	if result.Stale {
		t.Fatalf("expected project-a state to remain isolated, got %+v", result)
	}
}

func TestCheckStaleness(t *testing.T) {
	tests := []struct {
		name              string
		prepare           func(t *testing.T)
		currentNames      []string
		currentContents   []string
		expectStale       bool
		expectNew         []string
		expectRemoved     []string
		expectLastSyncSet bool
	}{
		{
			name: "never synced always stale",
			prepare: func(t *testing.T) {
				t.Helper()
			},
			currentNames:    []string{"slack", "github"},
			currentContents: []string{"b", "a"},
			expectStale:     true,
			expectNew:       []string{"github", "slack"},
			expectRemoved:   []string{},
		},
		{
			name: "synced then no change not stale",
			prepare: func(t *testing.T) {
				t.Helper()
				if err := RecordSync("project-a", []string{"github", "slack"}, []string{"a", "b"}); err != nil {
					t.Fatalf("record sync: %v", err)
				}
			},
			currentNames:      []string{"github", "slack"},
			currentContents:   []string{"a", "b"},
			expectStale:       false,
			expectNew:         []string{},
			expectRemoved:     []string{},
			expectLastSyncSet: true,
		},
		{
			name: "synced then new service added",
			prepare: func(t *testing.T) {
				t.Helper()
				if err := RecordSync("project-a", []string{"github"}, []string{"a"}); err != nil {
					t.Fatalf("record sync: %v", err)
				}
			},
			currentNames:      []string{"github", "slack"},
			currentContents:   []string{"a", "b"},
			expectStale:       true,
			expectNew:         []string{"slack"},
			expectRemoved:     []string{},
			expectLastSyncSet: true,
		},
		{
			name: "synced then service removed",
			prepare: func(t *testing.T) {
				t.Helper()
				if err := RecordSync("project-a", []string{"github", "slack"}, []string{"a", "b"}); err != nil {
					t.Fatalf("record sync: %v", err)
				}
			},
			currentNames:      []string{"github"},
			currentContents:   []string{"a"},
			expectStale:       true,
			expectNew:         []string{},
			expectRemoved:     []string{"slack"},
			expectLastSyncSet: true,
		},
		{
			name: "synced then content changed same names",
			prepare: func(t *testing.T) {
				t.Helper()
				if err := RecordSync("project-a", []string{"github"}, []string{"v1"}); err != nil {
					t.Fatalf("record sync: %v", err)
				}
			},
			currentNames:      []string{"github"},
			currentContents:   []string{"v2"},
			expectStale:       true,
			expectNew:         []string{},
			expectRemoved:     []string{},
			expectLastSyncSet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", tmp)
			t.Setenv("HOME", tmp)

			tt.prepare(t)

			result, err := CheckStaleness("project-a", tt.currentNames, tt.currentContents)
			if err != nil {
				t.Fatalf("check staleness: %v", err)
			}
			if result.Stale != tt.expectStale {
				t.Fatalf("expected stale=%v, got %v", tt.expectStale, result.Stale)
			}
			if !reflect.DeepEqual(result.NewServices, tt.expectNew) {
				t.Fatalf("unexpected new services: %+v", result.NewServices)
			}
			if !reflect.DeepEqual(result.RemovedServices, tt.expectRemoved) {
				t.Fatalf("unexpected removed services: %+v", result.RemovedServices)
			}
			if tt.expectLastSyncSet && result.LastSync == "" {
				t.Fatal("expected last sync to be set")
			}
			if !tt.expectLastSyncSet && result.LastSync != "" {
				t.Fatalf("expected empty last sync, got %q", result.LastSync)
			}
		})
	}
}

func TestFormatStaleWarning(t *testing.T) {
	tests := []struct {
		name     string
		result   *StaleCheckResult
		empty    bool
		contains []string
	}{
		{
			name: "stale with changes",
			result: &StaleCheckResult{
				Stale:           true,
				NewServices:     []string{"notion"},
				RemovedServices: []string{"slack"},
			},
			contains: []string{
				"warning: agent services out of sync",
				"+ notion (new)",
				"- slack (removed)",
				"Run: kimbap agents sync",
			},
		},
		{
			name: "content only change shows reason",
			result: &StaleCheckResult{
				Stale:           true,
				NewServices:     []string{},
				RemovedServices: []string{},
			},
			contains: []string{
				"warning: agent services out of sync",
				"service content changed",
				"Run: kimbap agents sync",
			},
		},
		{
			name:   "not stale returns empty",
			result: &StaleCheckResult{Stale: false},
			empty:  true,
		},
		{
			name:  "nil result returns empty",
			empty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning := FormatStaleWarning(tt.result)
			if tt.empty {
				if warning != "" {
					t.Fatalf("expected empty warning, got %q", warning)
				}
				return
			}

			for _, want := range tt.contains {
				if !strings.Contains(warning, want) {
					t.Fatalf("expected warning to contain %q, got %q", want, warning)
				}
			}
		})
	}
}

func TestToStringSet(t *testing.T) {
	set := toStringSet([]string{"github", "slack", "github"})
	if len(set) != 2 {
		t.Fatalf("expected set size 2, got %d", len(set))
	}
	if !set["github"] {
		t.Fatal("expected github key present")
	}
	if !set["slack"] {
		t.Fatal("expected slack key present")
	}
}

func TestSortByName(t *testing.T) {
	names := []string{"slack", "github", "notion"}
	contents := []string{"s-content", "g-content", "n-content"}

	sortedNames, sortedContents, err := sortByName(names, contents)
	if err != nil {
		t.Fatalf("sortByName: %v", err)
	}

	if !reflect.DeepEqual(sortedNames, []string{"github", "notion", "slack"}) {
		t.Fatalf("unexpected sorted names: %+v", sortedNames)
	}
	if !reflect.DeepEqual(sortedContents, []string{"g-content", "n-content", "s-content"}) {
		t.Fatalf("unexpected sorted contents: %+v", sortedContents)
	}
}

func TestSyncStatePathUsesXDGConfigHome(t *testing.T) {
	base := t.TempDir()
	xdg := filepath.Join(base, "my-xdg")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("HOME", filepath.Join(base, "other-home"))

	path, err := syncStatePath("project-a")
	if err != nil {
		t.Fatalf("syncStatePath: %v", err)
	}
	sum := sha256.Sum256([]byte("project-a"))
	expected := filepath.Join(xdg, "kimbap", "sync-state", hex.EncodeToString(sum[:])+".yaml")
	if path != expected {
		t.Fatalf("expected %q, got %q", expected, path)
	}
}
