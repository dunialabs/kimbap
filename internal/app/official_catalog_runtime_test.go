package app

import (
	"context"
	"runtime"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/config"
	runtimepkg "github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/skills"
)

func TestBuildRuntimeRegistersAllOfficialCatalogActions(t *testing.T) {
	ctx := context.Background()
	servicesDir := t.TempDir()
	installer := services.NewLocalInstaller(servicesDir)

	official, err := skills.List()
	if err != nil {
		t.Fatalf("list official skills: %v", err)
	}
	if len(official) < 50 {
		t.Fatalf("expected at least 50 official skills, got %d", len(official))
	}

	expected := map[string]struct{}{}
	for _, name := range official {
		data, getErr := skills.Get(name)
		if getErr != nil {
			t.Fatalf("get official skill %q: %v", name, getErr)
		}
		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			t.Fatalf("parse official skill %q: %v", name, parseErr)
		}
		if _, installErr := installer.Install(manifest, "official:"+name); installErr != nil {
			t.Fatalf("install official skill %q: %v", name, installErr)
		}
		for actionKey := range manifest.Actions {
			actionName := manifest.Name + "." + actionKey
			if _, exists := expected[actionName]; exists {
				t.Fatalf("duplicate action name in official catalog: %q", actionName)
			}
			expected[actionName] = struct{}{}
		}
	}

	rt, err := BuildRuntime(RuntimeDeps{Config: &config.KimbapConfig{Services: config.ServicesConfig{Dir: servicesDir}}})
	if err != nil {
		t.Fatalf("build runtime: %v", err)
	}

	listed, err := rt.ActionRegistry.List(ctx, runtimepkg.ListOptions{})
	if err != nil {
		t.Fatalf("list runtime actions: %v", err)
	}
	if len(listed) != len(expected) {
		t.Fatalf("runtime list count mismatch: expected=%d listed=%d", len(expected), len(listed))
	}

	actual := map[string]struct{}{}
	for _, def := range listed {
		if _, exists := actual[def.Name]; exists {
			t.Fatalf("duplicate runtime action entry %q", def.Name)
		}
		actual[def.Name] = struct{}{}
	}

	if len(actual) != len(expected) {
		t.Fatalf("runtime action count mismatch: expected=%d actual=%d", len(expected), len(actual))
	}

	for name := range expected {
		if _, ok := actual[name]; !ok {
			t.Fatalf("missing runtime action %q", name)
		}
	}
	for name := range actual {
		if _, ok := expected[name]; !ok {
			t.Fatalf("unexpected runtime action %q", name)
		}
	}

	validated := 0
	skippedAppleScript := 0
	for name := range expected {
		resolved, lookupErr := rt.ActionRegistry.Lookup(ctx, name)
		if lookupErr != nil {
			t.Fatalf("lookup action %q: %v", name, lookupErr)
		}
		if resolved == nil {
			t.Fatalf("lookup action %q returned nil definition", name)
		}

		adapterType := strings.TrimSpace(resolved.Adapter.Type)
		adapter, ok := rt.Adapters[adapterType]
		if !ok {
			if adapterType == "applescript" && runtime.GOOS != "darwin" {
				skippedAppleScript++
				continue
			}
			t.Fatalf("adapter %q for action %q is not registered", adapterType, name)
		}
		if validateErr := adapter.Validate(*resolved); validateErr != nil {
			t.Fatalf("adapter validate failed for %q: %v", name, validateErr)
		}
		validated++
	}

	if validated == 0 {
		t.Fatal("expected at least one validated action")
	}
	if runtime.GOOS != "darwin" && skippedAppleScript == 0 {
		t.Fatal("expected applescript actions to be skipped on non-darwin")
	}
}
