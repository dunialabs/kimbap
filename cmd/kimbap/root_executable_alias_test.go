package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestRewriteArgsForConfiguredExecutableAlias_CommandAliasBinary(t *testing.T) {
	in := []string{"/usr/local/bin/geosearch", "--name", "San Francisco"}
	aliases := map[string]string{"geosearch": "open-meteo-geocoding.search"}
	got := rewriteArgsForConfiguredExecutableAlias(in, aliases)
	want := []string{"/usr/local/bin/geosearch", "call", "open-meteo-geocoding.search", "--name", "San Francisco"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rewrite mismatch\nwant=%v\ngot =%v", want, got)
	}
}

func TestRewriteArgsForConfiguredExecutableAlias_KimbapBinaryUnchanged(t *testing.T) {
	in := []string{"/usr/local/bin/kimbap", "call", "open-meteo-geocoding.search"}
	aliases := map[string]string{"geosearch": "open-meteo-geocoding.search"}
	got := rewriteArgsForConfiguredExecutableAlias(in, aliases)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("expected kimbap binary args unchanged\nwant=%v\ngot =%v", in, got)
	}
}

func TestRewriteArgsForConfiguredExecutableAlias_InvalidTargetUnchanged(t *testing.T) {
	in := []string{"/usr/local/bin/geosearch", "--name", "San Francisco"}
	aliases := map[string]string{"geosearch": "open-meteo-geocoding"}
	got := rewriteArgsForConfiguredExecutableAlias(in, aliases)
	if !reflect.DeepEqual(got, in) {
		t.Fatalf("expected invalid target alias to be ignored\nwant=%v\ngot =%v", in, got)
	}
}

func TestRewriteArgsForExecutableAliasUsesDataDirConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))
	t.Setenv("KIMBAP_CONFIG", "")

	dataDir := t.TempDir()
	configPath := filepath.Join(dataDir, "config.yaml")
	configBody := []byte("mode: embedded\ncommand_aliases:\n  geosearch: open-meteo-geocoding.search\n")
	if err := os.WriteFile(configPath, configBody, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	in := []string{"/usr/local/bin/geosearch", "--data-dir", dataDir, "--name", "San Francisco"}
	got := rewriteArgsForExecutableAlias(in)
	want := []string{"/usr/local/bin/geosearch", "call", "open-meteo-geocoding.search", "--data-dir", dataDir, "--name", "San Francisco"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rewrite mismatch\nwant=%v\ngot =%v", want, got)
	}
}

func TestRewriteArgsForExecutableAliasUsesKimbapConfigEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	configPath := filepath.Join(t.TempDir(), "env-config.yaml")
	configBody := []byte("mode: embedded\ncommand_aliases:\n  geosearch: open-meteo-geocoding.search\n")
	if err := os.WriteFile(configPath, configBody, 0o644); err != nil {
		t.Fatalf("write env config: %v", err)
	}
	t.Setenv("KIMBAP_CONFIG", configPath)

	in := []string{"/usr/local/bin/geosearch", "--name", "San Francisco"}
	got := rewriteArgsForExecutableAlias(in)
	want := []string{"/usr/local/bin/geosearch", "call", "open-meteo-geocoding.search", "--name", "San Francisco"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rewrite mismatch\nwant=%v\ngot =%v", want, got)
	}
}

func TestParseConfigPathAndDataDirArgs(t *testing.T) {
	configPath, dataDir := parseConfigPathAndDataDirArgs([]string{
		"kimbap", "geosearch", "--config=/tmp/c.yaml", "--data-dir", "/tmp/data", "--", "--config", "ignored",
	})
	if configPath != "/tmp/c.yaml" {
		t.Fatalf("expected config path /tmp/c.yaml, got %q", configPath)
	}
	if dataDir != "/tmp/data" {
		t.Fatalf("expected data dir /tmp/data, got %q", dataDir)
	}
}

func TestRewriteArgsForExecutableAliasDataDirOverridesKimbapConfigEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg"))

	envConfigPath := filepath.Join(t.TempDir(), "env-config.yaml")
	if err := os.WriteFile(envConfigPath, []byte("mode: embedded\ncommand_aliases:\n  geosearch: env.noop\n"), 0o644); err != nil {
		t.Fatalf("write env config: %v", err)
	}
	t.Setenv("KIMBAP_CONFIG", envConfigPath)

	dataDir := t.TempDir()
	dataConfigPath := filepath.Join(dataDir, "config.yaml")
	if err := os.WriteFile(dataConfigPath, []byte("mode: embedded\ncommand_aliases:\n  geosearch: data.noop\n"), 0o644); err != nil {
		t.Fatalf("write data-dir config: %v", err)
	}

	in := []string{"/usr/local/bin/geosearch", "--data-dir", dataDir, "--name", "San Francisco"}
	got := rewriteArgsForExecutableAlias(in)
	want := []string{"/usr/local/bin/geosearch", "call", "data.noop", "--data-dir", dataDir, "--name", "San Francisco"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rewrite mismatch\nwant=%v\ngot =%v", want, got)
	}
}
