package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dunialabs/kimbap/internal/actions"
)

func TestCommandAdapterType(t *testing.T) {
	a := NewCommandAdapter(nil, time.Second)
	if got := a.Type(); got != "command" {
		t.Fatalf("Type() = %q, want command", got)
	}
}

func TestCommandAdapterValidate(t *testing.T) {
	a := NewCommandAdapter(nil, time.Second)
	if err := a.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{ExecutablePath: "/bin/echo", Command: "ping"}}); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
	if err := a.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{Command: "ping"}}); err == nil {
		t.Fatal("expected missing executable validation error")
	}
	if err := a.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{ExecutablePath: "/bin/echo"}}); err == nil {
		t.Fatal("expected missing command validation error")
	}
}

func TestCommandAdapterValidate_Allowlist(t *testing.T) {
	a := NewCommandAdapter([]string{"/bin/echo"}, time.Second)
	if err := a.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{ExecutablePath: "/usr/bin/env", Command: "ping"}}); err == nil {
		t.Fatal("expected allowlist validation error")
	}
}

func TestCommandAdapterExecute_JSONAndArgsAndEnv(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Auth: actions.AuthRequirement{CredentialRef: "diagram.token"},
			InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
				"count": {Type: "integer"},
				"title": {Type: "string"},
				"empty": {Type: "string"},
			}},
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- subcmd create",
				JSONFlag:       "--json",
				EnvInject:      map[string]string{"STATIC_ENV": "ok", "GO_WANT_HELPER_PROCESS": "1"},
			},
		},
		Input: map[string]any{"title": "diagram", "count": 2, "empty": "  "},
		Credentials: &actions.ResolvedCredentialSet{
			Token: "secret-token",
		},
	})

	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}

	argsAny, ok := res.Output["args"].([]any)
	if !ok {
		t.Fatalf("output[args] type = %T, want []any", res.Output["args"])
	}
	gotArgs := make([]string, 0, len(argsAny))
	for _, item := range argsAny {
		gotArgs = append(gotArgs, fmt.Sprintf("%v", item))
	}
	joined := strings.Join(gotArgs, " ")
	for _, want := range []string{"subcmd", "create", "--count", "2", "--title", "diagram", "--json"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected args to contain %q, got %v", want, gotArgs)
		}
	}
	if strings.Contains(joined, "--empty") {
		t.Fatalf("did not expect --empty arg, got %v", gotArgs)
	}
	if res.Output["cred"] != "secret-token" {
		t.Fatalf("credential env not injected, got %v", res.Output["cred"])
	}
	if res.Output["static"] != "ok" {
		t.Fatalf("static env not injected, got %v", res.Output["static"])
	}
}

func TestCommandAdapterExecute_InvalidJSONFallsBackToRaw(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- raw-output",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.Output["raw"] != "plain text output" {
		t.Fatalf("output[raw] = %v, want plain text output", res.Output["raw"])
	}
}

func TestCommandAdapterExecute_DoesNotForwardUndeclaredInputFields(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
				"title": {Type: "string"},
			}},
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- subcmd create",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
		Input: map[string]any{"title": "diagram", "inject": "--danger"},
	})

	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	argsAny, ok := res.Output["args"].([]any)
	if !ok {
		t.Fatalf("output[args] type = %T, want []any", res.Output["args"])
	}
	joined := ""
	for _, item := range argsAny {
		joined += " " + fmt.Sprintf("%v", item)
	}
	if strings.Contains(joined, "--inject") {
		t.Fatalf("expected undeclared input not to be forwarded, got %v", argsAny)
	}
	if !strings.Contains(joined, "--title") {
		t.Fatalf("expected declared input to be forwarded, got %v", argsAny)
	}
}

func TestCommandAdapterExecute_AllowlistRejected(t *testing.T) {
	a := NewCommandAdapter([]string{"/bin/echo"}, time.Second)
	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{ExecutablePath: os.Args[0], Command: "whatever"},
		},
	})
	if err == nil {
		t.Fatal("expected allowlist rejection error")
	}
	if res == nil || res.HTTPStatus != 403 {
		t.Fatalf("expected HTTP 403, got %+v", res)
	}
}

func TestCommandAdapterExecute_CommandFailureIncludesStderr(t *testing.T) {
	a := NewCommandAdapter(nil, time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{ExecutablePath: os.Args[0], Command: "-test.run=TestCommandAdapterHelperProcess -- stderr-fail", EnvInject: map[string]string{"GO_WANT_HELPER_PROCESS": "1"}},
		},
	})
	if err == nil {
		t.Fatal("expected execution error")
	}
	if res == nil {
		t.Fatal("expected result on failure")
	}
	if !strings.Contains(actions.AsExecutionError(err).Message, "boom from helper") {
		t.Fatalf("expected stderr in error message, got %v", err)
	}
}

func TestCommandAdapterHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	idx := -1
	for i, v := range args {
		if v == "--" {
			idx = i
			break
		}
	}
	forwarded := []string{}
	if idx >= 0 && idx+1 < len(args) {
		forwarded = args[idx+1:]
	}

	if len(forwarded) > 0 {
		switch forwarded[0] {
		case "raw-output":
			_, _ = os.Stdout.WriteString("plain text output")
			os.Exit(0)
		case "stderr-fail":
			_, _ = os.Stderr.WriteString("boom from helper")
			os.Exit(1)
		}
	}

	payload := map[string]any{
		"args":   forwarded,
		"cred":   os.Getenv("KIMBAP_CREDENTIAL_DIAGRAM_TOKEN"),
		"static": os.Getenv("STATIC_ENV"),
	}
	encoded, _ := json.Marshal(payload)
	_, _ = os.Stdout.Write(encoded)
	os.Exit(0)
}

func TestSafeProcessEnv_AllowlistOnly(t *testing.T) {
	t.Setenv("PATH", "/usr/bin")
	t.Setenv("LANG", "en_US.UTF-8")
	t.Setenv("LC_ALL", "en_US.UTF-8")
	t.Setenv("HTTP_PROXY", "http://proxy.local:8080")
	t.Setenv("KIMBAP_MASTER_KEY_HEX", "secret")

	out := safeProcessEnv()
	joined := strings.Join(out, "\n")

	for _, want := range []string{"PATH=/usr/bin", "LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q to be preserved, got %v", want, out)
		}
	}

	for _, blocked := range []string{"HTTP_PROXY=http://proxy.local:8080", "KIMBAP_MASTER_KEY_HEX=secret"} {
		if strings.Contains(joined, blocked) {
			t.Fatalf("expected %q to be excluded by allowlist, got %v", blocked, out)
		}
	}
}
