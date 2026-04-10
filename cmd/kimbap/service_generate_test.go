package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dunialabs/kimbap/internal/services"
)

func TestServiceGenerateRejectsUppercaseHTTPSource(t *testing.T) {
	resetOptsForTest(t)

	cmd := newServiceGenerateCommand()
	cmd.SetArgs([]string{"--openapi", "HTTP://example.com/openapi.yaml"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected insecure HTTP source to be rejected")
	}
	if !strings.Contains(err.Error(), "insecure URL") {
		t.Fatalf("expected insecure URL error, got %v", err)
	}
}

func TestServiceGenerateAllowsLoopbackHTTPSource(t *testing.T) {
	resetOptsForTest(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`openapi: 3.0.0
info:
  title: Loopback CLI API
  version: 1.0.0
servers:
  - url: /api
paths:
  /health:
    get:
      operationId: health
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`))
	}))
	defer server.Close()

	cmd := newServiceGenerateCommand()
	cmd.SetArgs([]string{"--openapi", server.URL})
	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("service generate from loopback HTTP failed: %v", err)
	}

	manifest, err := services.ParseManifest([]byte(output))
	if err != nil {
		t.Fatalf("parse generated manifest: %v\noutput=%s", err, output)
	}
	if manifest.Name != "loopback-cli-api" {
		t.Fatalf("unexpected generated service name: %q", manifest.Name)
	}
	if manifest.BaseURL != server.URL+"/api" {
		t.Fatalf("expected relative server URL to resolve, got %q", manifest.BaseURL)
	}
}

func TestServiceGenerateAppliesNameTagAndPathPrefixFilters(t *testing.T) {
	resetOptsForTest(t)
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(specPath, []byte(serviceGenerateFilterFixture), 0o644); err != nil {
		t.Fatalf("write OpenAPI fixture: %v", err)
	}

	cmd := newServiceGenerateCommand()
	cmd.SetArgs([]string{
		"--openapi", specPath,
		"--name", "Generated Admin API",
		"--tag", "ADMIN",
		"--path-prefix", "admin",
	})
	output, err := captureStdout(t, cmd.Execute)
	if err != nil {
		t.Fatalf("service generate with filters failed: %v", err)
	}

	manifest, err := services.ParseManifest([]byte(output))
	if err != nil {
		t.Fatalf("parse filtered generated manifest: %v\noutput=%s", err, output)
	}
	if manifest.Name != "generated-admin-api" {
		t.Fatalf("expected normalized name override, got %q", manifest.Name)
	}
	if len(manifest.Actions) != 2 {
		t.Fatalf("expected 2 filtered actions, got %+v", manifest.Actions)
	}
	if _, ok := manifest.Actions["listusers"]; !ok {
		t.Fatalf("expected admin listusers action, got %+v", manifest.Actions)
	}
	if _, ok := manifest.Actions["listaudit"]; !ok {
		t.Fatalf("expected admin listaudit action, got %+v", manifest.Actions)
	}
	for actionName := range manifest.Actions {
		if strings.HasPrefix(actionName, "listusers-") {
			t.Fatalf("expected filter to run before action key disambiguation, got %q", actionName)
		}
	}
}

func TestServiceGeneratePrintsWarningsToStderrInTextMode(t *testing.T) {
	resetOptsForTest(t)
	opts.format = "text"

	specPath := filepath.Join(t.TempDir(), "warning-openapi.yaml")
	if err := os.WriteFile(specPath, []byte(serviceGenerateWarningFixture), 0o644); err != nil {
		t.Fatalf("write warning OpenAPI fixture: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "generated.yaml")

	cmd := newServiceGenerateCommand()
	cmd.SetArgs([]string{"--openapi", specPath, "--output", outputPath})
	stderr, err := captureStderr(t, cmd.Execute)
	if err != nil {
		t.Fatalf("service generate warning fixture failed: %v", err)
	}
	if !strings.Contains(stderr, "warning: generated action upload:") {
		t.Fatalf("expected generated action warning in stderr, got %q", stderr)
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("expected generated manifest file, stat err=%v", err)
	}
}

func TestServiceGenerateInstallWritesFileAndInstalls(t *testing.T) {
	dataDir := t.TempDir()
	servicesDir := filepath.Join(dataDir, "services")
	configPath := writeServiceCLIConfig(t, dataDir, servicesDir)
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(specPath, []byte(serviceGenerateFilterFixture), 0o644); err != nil {
		t.Fatalf("write OpenAPI fixture: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "generated.yaml")

	withServiceCLIOpts(t, configPath, func() {
		cmd := newServiceGenerateCommand()
		cmd.SetArgs([]string{
			"--openapi", specPath,
			"--name", "Generated Admin API",
			"--tag", "admin",
			"--path-prefix", "/admin",
			"--output", outputPath,
			"--install",
		})
		output, err := captureStdout(t, cmd.Execute)
		if err != nil {
			t.Fatalf("service generate --install failed: %v", err)
		}

		payload := decodeJSONObject(t, output)
		if generated, _ := payload["generated"].(bool); !generated {
			t.Fatalf("expected generated=true payload, got %+v", payload)
		}
		if installed, _ := payload["installed"].(bool); !installed {
			t.Fatalf("expected installed=true payload, got %+v", payload)
		}
		if gotOutputPath, _ := payload["output_path"].(string); gotOutputPath != outputPath {
			t.Fatalf("expected output_path=%q, got %+v", outputPath, payload)
		}

		cfg, err := loadAppConfig()
		if err != nil {
			t.Fatalf("loadAppConfig() error: %v", err)
		}
		installed, err := installerFromConfig(cfg).Get("generated-admin-api")
		if err != nil {
			t.Fatalf("installer.Get() error: %v", err)
		}
		if installed.Source != "local:"+outputPath {
			t.Fatalf("installed source = %q, want %q", installed.Source, "local:"+outputPath)
		}
		if len(installed.Manifest.Actions) != 2 {
			t.Fatalf("expected installed generated manifest to keep 2 actions, got %+v", installed.Manifest.Actions)
		}
	})
}

func TestServiceGenerateRejectsSymlinkOutputPath(t *testing.T) {
	resetOptsForTest(t)
	specPath := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(specPath, []byte(serviceGenerateFilterFixture), 0o644); err != nil {
		t.Fatalf("write OpenAPI fixture: %v", err)
	}
	base := t.TempDir()
	realTarget := filepath.Join(base, "real.yaml")
	outputPath := filepath.Join(base, "generated.yaml")
	if err := os.WriteFile(realTarget, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write real target: %v", err)
	}
	if err := os.Symlink(realTarget, outputPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	cmd := newServiceGenerateCommand()
	cmd.SetArgs([]string{"--openapi", specPath, "--output", outputPath})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected symlink output path to be rejected")
	}
	if !strings.Contains(err.Error(), "symlinked output path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsServiceHTTPURLHandlesUppercaseScheme(t *testing.T) {
	if !isServiceHTTPURL("HTTPS://example.com/openapi.yaml") {
		t.Fatal("expected uppercase HTTPS scheme to be recognized")
	}
	if !isServiceHTTPURL("HTTP://example.com/openapi.yaml") {
		t.Fatal("expected uppercase HTTP scheme to be recognized")
	}
}

const serviceGenerateFilterFixture = `openapi: 3.0.3
info:
  title: Filter Fixture API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /public/users:
    get:
      operationId: listUsers
      tags: [public]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /admin/users:
    get:
      operationId: listUsers
      tags: [admin]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
  /admin/audit:
    get:
      operationId: listAudit
      tags: [admin]
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`

const serviceGenerateWarningFixture = `openapi: 3.0.3
info:
  title: Warning Fixture API
  version: 1.0.0
servers:
  - url: https://api.example.com
paths:
  /upload:
    post:
      operationId: upload
      requestBody:
        required: true
        content:
          application/xml:
            schema:
              type: string
      responses:
        '200':
          description: ok
          content:
            application/json:
              schema:
                type: object
`
