package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func withServiceCLITextOpts(t *testing.T, configPath string, fn func()) {
	t.Helper()
	prev := opts
	opts = cliOptions{configPath: configPath, format: "text", noSplash: true}
	t.Cleanup(func() { opts = prev })
	fn()
}

func TestServiceCLIListAvailableIncludesCatalogMetadata(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		listCmd := newServiceListCommand()
		listCmd.SetArgs([]string{"--available"})
		output, err := captureStdout(t, listCmd.Execute)
		if err != nil {
			t.Fatalf("service list --available failed: %v", err)
		}

		rows := decodeJSONArrayOfObjects(t, output)
		var githubRow map[string]any
		for _, row := range rows {
			if row["name"] == "github" {
				githubRow = row
				break
			}
		}
		if githubRow == nil {
			t.Fatal("github row missing from --available output")
		}
		if description, _ := githubRow["description"].(string); !strings.Contains(description, "GitHub REST API integration") {
			t.Fatalf("expected github description in row, got %+v", githubRow)
		}
		if adapter, _ := githubRow["adapter"].(string); adapter != "http" {
			t.Fatalf("expected adapter=http, got %+v", githubRow)
		}
		if actions, _ := githubRow["actions"].(float64); actions != 3 {
			t.Fatalf("expected actions=3, got %+v", githubRow)
		}
		if authRequired, _ := githubRow["auth_required"].(bool); !authRequired {
			t.Fatalf("expected auth_required=true, got %+v", githubRow)
		}
		triggers, ok := githubRow["triggers"].(map[string]any)
		if !ok {
			t.Fatalf("expected triggers object, got %+v", githubRow)
		}
		taskVerbs, ok := triggers["task_verbs"].([]any)
		if !ok || !anySliceContainsString(taskVerbs, "inspect") {
			t.Fatalf("expected trigger task_verbs to include inspect, got %+v", triggers)
		}
	})
}

func TestServiceSearchFindsCatalogMatchesAcrossMetadata(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		t.Run("trigger metadata", func(t *testing.T) {
			cmd := newServiceSearchCommand()
			cmd.SetArgs([]string{"github apis directly"})
			output, err := captureStdout(t, cmd.Execute)
			if err != nil {
				t.Fatalf("service search failed: %v", err)
			}

			var results []catalogSearchResult
			if err := json.Unmarshal([]byte(output), &results); err != nil {
				t.Fatalf("decode search output: %v\noutput=%s", err, output)
			}
			if len(results) == 0 || results[0].Name != "github" {
				t.Fatalf("expected github to rank for trigger metadata search, got %+v", results)
			}
			if !slicesContains(results[0].MatchedFields, "triggers") {
				t.Fatalf("expected trigger field match, got %+v", results[0])
			}
		})

		t.Run("action name", func(t *testing.T) {
			cmd := newServiceSearchCommand()
			cmd.SetArgs([]string{"create-issue"})
			output, err := captureStdout(t, cmd.Execute)
			if err != nil {
				t.Fatalf("service search failed: %v", err)
			}

			var results []catalogSearchResult
			if err := json.Unmarshal([]byte(output), &results); err != nil {
				t.Fatalf("decode search output: %v\noutput=%s", err, output)
			}
			if len(results) == 0 || results[0].Name != "github" {
				t.Fatalf("expected github for action-name search, got %+v", results)
			}
			if !slicesContains(results[0].MatchedActions, "create-issue") {
				t.Fatalf("expected matched action create-issue, got %+v", results[0])
			}
		})
	})
}

func TestServiceSearchLimitAndEmptyResults(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cmd := newServiceSearchCommand()
		cmd.SetArgs([]string{"open-meteo", "--limit", "2"})
		output, err := captureStdout(t, cmd.Execute)
		if err != nil {
			t.Fatalf("service search --limit failed: %v", err)
		}

		var results []catalogSearchResult
		if err := json.Unmarshal([]byte(output), &results); err != nil {
			t.Fatalf("decode limited search output: %v\noutput=%s", err, output)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %+v", results)
		}
		if results[0].Name != "open-meteo" || results[1].Name != "open-meteo-air-quality" {
			t.Fatalf("expected deterministic first two results, got %+v", results)
		}

		emptyCmd := newServiceSearchCommand()
		emptyCmd.SetArgs([]string{"definitely-no-catalog-match"})
		emptyOutput, err := captureStdout(t, emptyCmd.Execute)
		if err != nil {
			t.Fatalf("service search empty failed: %v", err)
		}
		if strings.TrimSpace(emptyOutput) != "[]" {
			t.Fatalf("expected empty JSON array, got %q", emptyOutput)
		}
	})

	withServiceCLITextOpts(t, configPath, func() {
		cmd := newServiceSearchCommand()
		cmd.SetArgs([]string{"definitely-no-catalog-match"})
		output, err := captureStdout(t, cmd.Execute)
		if err != nil {
			t.Fatalf("service search text empty failed: %v", err)
		}
		if !strings.Contains(output, "No matching catalog services found.") {
			t.Fatalf("expected empty text message, got %q", output)
		}
	})
}

func TestServiceDescribeOutputsCatalogSummary(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cmd := newServiceDescribeCommand()
		cmd.SetArgs([]string{"github"})
		output, err := captureStdout(t, cmd.Execute)
		if err != nil {
			t.Fatalf("service describe failed: %v", err)
		}

		var payload catalogDescribePayload
		if err := json.Unmarshal([]byte(output), &payload); err != nil {
			t.Fatalf("decode describe output: %v\noutput=%s", err, output)
		}
		if payload.Name != "github" || payload.Adapter != "http" {
			t.Fatalf("unexpected describe payload: %+v", payload)
		}
		if payload.AuthType != "bearer" || !payload.AuthRequired {
			t.Fatalf("expected bearer auth summary, got %+v", payload)
		}
		if payload.ActionCount != 3 || len(payload.Actions) != 3 {
			t.Fatalf("expected 3 actions, got %+v", payload)
		}
		if !strings.Contains(payload.InstallHint, "kimbap service install github") {
			t.Fatalf("expected install hint, got %+v", payload)
		}
	})

	withServiceCLITextOpts(t, configPath, func() {
		cmd := newServiceDescribeCommand()
		cmd.SetArgs([]string{"github"})
		output, err := captureStdout(t, cmd.Execute)
		if err != nil {
			t.Fatalf("service describe text failed: %v", err)
		}
		if !strings.Contains(output, "Actions (3):") {
			t.Fatalf("expected action summary in text output, got %q", output)
		}
		if !strings.Contains(output, "create-issue") {
			t.Fatalf("expected action name in text output, got %q", output)
		}
		if !strings.Contains(output, "Run 'kimbap service install github'") {
			t.Fatalf("expected install hint in text output, got %q", output)
		}
	})
}

func TestServiceDescribeUnknownServiceSuggestsCatalogEntry(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)

	withServiceCLIOpts(t, configPath, func() {
		cmd := newServiceDescribeCommand()
		cmd.SetArgs([]string{"githb"})
		_, err := captureStdout(t, cmd.Execute)
		if err == nil {
			t.Fatal("expected unknown service error")
		}
		if !strings.Contains(err.Error(), `Did you mean "github"?`) {
			t.Fatalf("expected suggestion in error, got %q", err.Error())
		}
	})
}

func TestServiceDiscoveryCommandsDoNotMaterializeDataDir(t *testing.T) {
	servicesDir := t.TempDir()
	missingDataDir := filepath.Join(t.TempDir(), "missing-data-dir")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeMinimalConfig(t, configPath, missingDataDir, servicesDir)

	prev := opts
	opts = cliOptions{configPath: configPath, format: "json", noSplash: true}
	t.Cleanup(func() { opts = prev })

	commands := []*cobra.Command{
		newServiceListCommand(),
		newServiceSearchCommand(),
		newServiceDescribeCommand(),
	}
	commands[0].SetArgs([]string{"--available"})
	commands[1].SetArgs([]string{"github"})
	commands[2].SetArgs([]string{"github"})

	for _, cmd := range commands {
		if _, err := captureStdout(t, cmd.Execute); err != nil {
			t.Fatalf("%s failed: %v", cmd.Name(), err)
		}
	}

	if _, err := os.Stat(missingDataDir); !os.IsNotExist(err) {
		t.Fatalf("service discovery commands must not create data_dir, stat err=%v", err)
	}
}

func slicesContains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
