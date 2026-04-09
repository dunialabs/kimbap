package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	if err := a.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{ExecutablePath: "/bin/echo"}}); err != nil {
		t.Fatalf("Validate() with empty command error = %v, want nil", err)
	}
	if err := a.Validate(actions.ActionDefinition{Adapter: actions.AdapterConfig{}}); err == nil {
		t.Fatal("expected error when both executable and command are empty")
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

func TestCommandAdapterExecute_JSONFlagDisabled(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- subcmd create",
				JSONFlag:       "none",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	argsAny, ok := res.Output["args"].([]any)
	if !ok {
		t.Fatalf("output[args] type = %T, want []any", res.Output["args"])
	}
	var joinedBuilder strings.Builder
	for _, item := range argsAny {
		_, _ = fmt.Fprintf(&joinedBuilder, " %v", item)
	}
	joined := joinedBuilder.String()
	if strings.Contains(joined, "--json") {
		t.Fatalf("did not expect --json when json_flag disabled, got %v", argsAny)
	}
}

func TestCommandAdapterExecute_JSONFlagSupportsMultipleTokens(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- subcmd create",
				JSONFlag:       "--output json",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	argsAny, ok := res.Output["args"].([]any)
	if !ok {
		t.Fatalf("output[args] type = %T, want []any", res.Output["args"])
	}
	var joinedBuilder strings.Builder
	for _, item := range argsAny {
		_, _ = fmt.Fprintf(&joinedBuilder, " %v", item)
	}
	joined := joinedBuilder.String()
	for _, want := range []string{"--output", "json"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected args to contain %q, got %v", want, argsAny)
		}
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

func TestCommandAdapterExecute_ExitCodePreserved(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- exit-code 0",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if got := res.Output["_exit_code"]; got != float64(0) {
		t.Fatalf("output[_exit_code] = %#v, want 0", got)
	}
	if res.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", res.ExitCode)
	}
}

func TestCommandAdapterExecute_SuccessCodes(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- exit-code 1",
				SuccessCodes:   []int{0, 1},
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}
	if got := res.Output["_exit_code"]; got != float64(1) {
		t.Fatalf("output[_exit_code] = %#v, want 1", got)
	}
	if res.ExitCode != 1 {
		t.Fatalf("ExitCode = %d, want 1", res.ExitCode)
	}
}

func TestCommandAdapterExecute_ExitCodeInError(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- exit-code 127",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected execution error")
	}
	if res == nil {
		t.Fatal("expected result on failure")
	}
	if res.HTTPStatus != 502 {
		t.Fatalf("HTTPStatus = %d, want 502", res.HTTPStatus)
	}
	if got := res.Output["_exit_code"]; got != float64(127) {
		t.Fatalf("output[_exit_code] = %#v, want 127", got)
	}
	if res.ExitCode != 127 {
		t.Fatalf("ExitCode = %d, want 127", res.ExitCode)
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
		case "exit-code":
			if len(forwarded) > 1 {
				code, _ := strconv.Atoi(forwarded[1])
				os.Exit(code)
			}
			os.Exit(1)
		case "echo-args":
			payload := map[string]any{"args": forwarded}
			encoded, _ := json.Marshal(payload)
			_, _ = os.Stdout.Write(encoded)
			os.Exit(0)
		case "raw-output":
			_, _ = os.Stdout.WriteString("plain text output")
			os.Exit(0)
		case "stderr-fail":
			_, _ = os.Stderr.WriteString("boom from helper")
			os.Exit(1)
		case "ndjson-output":
			fmt.Fprintln(os.Stdout, `{"id":"abc","status":"Up"}`)
			fmt.Fprintln(os.Stdout, `{"id":"def","status":"Exited"}`)
			os.Exit(0)
		case "ndjson-mixed":
			fmt.Fprintln(os.Stdout, `{"id":"abc","status":"Up"}`)
			fmt.Fprintln(os.Stdout, `not json at all`)
			fmt.Fprintln(os.Stdout, `{"id":"def","status":"Exited"}`)
			os.Exit(0)
		case "stderr-merge":
			_, _ = os.Stdout.WriteString("OUT ")
			_, _ = os.Stderr.WriteString("ERR ")
			_, _ = os.Stdout.WriteString("DONE")
			os.Exit(0)
		case "stderr-merge-fail":
			_, _ = os.Stdout.WriteString("OUT ")
			_, _ = os.Stderr.WriteString("ERR ")
			_, _ = os.Stdout.WriteString("DONE")
			os.Exit(1)
		case "show-env":
			envMap := map[string]string{}
			for _, entry := range os.Environ() {
				parts := strings.SplitN(entry, "=", 2)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}
			encoded, _ := json.Marshal(map[string]any{"env": envMap, "args": forwarded})
			_, _ = os.Stdout.Write(encoded)
			os.Exit(0)
		case "large-success-output":
			_, _ = os.Stdout.WriteString(strings.Repeat("x", int(maxCommandOutputBytes)+1))
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

func TestCommandAdapterExecute_PositionalArgs(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			InputSchema: &actions.Schema{
				Type:     "object",
				ArgOrder: []string{"ref", "stat", "count"},
				Properties: map[string]*actions.Schema{
					"ref":   {Type: "string", ArgStyle: "positional"},
					"stat":  {Type: "boolean", ArgStyle: "boolean"},
					"count": {Type: "string", ArgStyle: "flag"},
				},
			},
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- echo-args",
				JSONFlag:       "none",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
		Input: map[string]any{"ref": "HEAD~3", "stat": true, "count": "5"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	argsAny, ok := res.Output["args"].([]any)
	if !ok {
		t.Fatalf("output[args] type = %T, want []any", res.Output["args"])
	}
	gotArgs := make([]string, 0, len(argsAny))
	for _, item := range argsAny {
		gotArgs = append(gotArgs, fmt.Sprintf("%v", item))
	}
	wantArgs := []string{"echo-args", "HEAD~3", "--stat", "--count", "5"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args length = %d, want %d (%v)", len(gotArgs), len(wantArgs), gotArgs)
	}
	for i, want := range wantArgs {
		if gotArgs[i] != want {
			t.Fatalf("args[%d] = %q, want %q (all args: %v)", i, gotArgs[i], want, gotArgs)
		}
	}
}

func TestCommandAdapterExecute_BooleanFalseOmitsFlag(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			InputSchema: &actions.Schema{
				Type:     "object",
				ArgOrder: []string{"verbose"},
				Properties: map[string]*actions.Schema{
					"verbose": {Type: "boolean", ArgStyle: "boolean"},
				},
			},
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- echo-args",
				JSONFlag:       "none",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
		Input: map[string]any{"verbose": false},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}

	argsAny, ok := res.Output["args"].([]any)
	if !ok {
		t.Fatalf("output[args] type = %T, want []any", res.Output["args"])
	}
	gotArgs := make([]string, 0, len(argsAny))
	for _, item := range argsAny {
		gotArgs = append(gotArgs, fmt.Sprintf("%v", item))
	}
	for _, arg := range gotArgs {
		if arg == "--verbose" {
			t.Fatalf("did not expect --verbose in args, got %v", gotArgs)
		}
	}
}

func TestCommandAdapterExecute_EmptyCommand(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			InputSchema: &actions.Schema{Properties: map[string]*actions.Schema{
				"greeting": {Type: "string"},
			}},
			Adapter: actions.AdapterConfig{
				ExecutablePath: "/bin/echo",
				Command:        "",
				JSONFlag:       "none",
			},
		},
		Input: map[string]any{"greeting": "hello"},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}
	raw, ok := res.Output["raw"].(string)
	if !ok {
		t.Fatalf("output[raw] type = %T, want string", res.Output["raw"])
	}
	if !strings.Contains(raw, "--greeting hello") {
		t.Fatalf("output[raw] = %q, want to contain '--greeting hello'", raw)
	}
}

func TestCommandAdapterExecute_EnvPassthrough(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	t.Setenv("KIMBAP_TEST_PASS_EXACT", "exact_val")
	t.Setenv("KIMBAP_TEST_PASS_PREFIX_A", "prefix_a_val")
	t.Setenv("KIMBAP_TEST_PASS_PREFIX_B", "prefix_b_val")
	t.Setenv("KIMBAP_TEST_NO_MATCH", "should_not_appear")

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- show-env",
				JSONFlag:       "none",
				EnvPassthrough: []string{"KIMBAP_TEST_PASS_EXACT", "KIMBAP_TEST_PASS_PREFIX_*"},
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}

	envAny, ok := res.Output["env"]
	if !ok {
		t.Fatal("expected env key in output")
	}
	envMap, ok := envAny.(map[string]any)
	if !ok {
		t.Fatalf("env type = %T, want map[string]any", envAny)
	}

	if envMap["KIMBAP_TEST_PASS_EXACT"] != "exact_val" {
		t.Fatalf("expected KIMBAP_TEST_PASS_EXACT=exact_val, got %v", envMap["KIMBAP_TEST_PASS_EXACT"])
	}
	if envMap["KIMBAP_TEST_PASS_PREFIX_A"] != "prefix_a_val" {
		t.Fatalf("expected KIMBAP_TEST_PASS_PREFIX_A=prefix_a_val, got %v", envMap["KIMBAP_TEST_PASS_PREFIX_A"])
	}
	if envMap["KIMBAP_TEST_PASS_PREFIX_B"] != "prefix_b_val" {
		t.Fatalf("expected KIMBAP_TEST_PASS_PREFIX_B=prefix_b_val, got %v", envMap["KIMBAP_TEST_PASS_PREFIX_B"])
	}
	if _, found := envMap["KIMBAP_TEST_NO_MATCH"]; found {
		t.Fatalf("KIMBAP_TEST_NO_MATCH should not be passed through, got %v", envMap["KIMBAP_TEST_NO_MATCH"])
	}
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

func TestCommandAdapterExecute_MergeStderr(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- stderr-merge",
				JSONFlag:       "none",
				MergeStderr:    true,
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}
	raw, ok := res.Output["raw"].(string)
	if !ok {
		t.Fatalf("output[raw] type = %T, want string", res.Output["raw"])
	}
	if !strings.Contains(raw, "OUT") {
		t.Fatalf("expected stdout content in merged output, got %q", raw)
	}
	if !strings.Contains(raw, "ERR") {
		t.Fatalf("expected stderr content in merged output, got %q", raw)
	}
}

func TestCommandAdapterExecute_MergeStderrFailureIncludesMergedRaw(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- stderr-merge-fail",
				JSONFlag:       "none",
				MergeStderr:    true,
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected execution error")
	}
	if res == nil {
		t.Fatal("expected result on failure")
	}
	raw, ok := res.Output["raw"].(string)
	if !ok {
		t.Fatalf("output[raw] type = %T, want string", res.Output["raw"])
	}
	if !strings.Contains(raw, "OUT ") || !strings.Contains(raw, "ERR ") || !strings.Contains(raw, "DONE") {
		t.Fatalf("expected merged raw diagnostics, got %q", raw)
	}
	if !strings.Contains(actions.AsExecutionError(err).Message, "OUT ERR DONE") {
		t.Fatalf("expected merged diagnostics in error message, got %v", err)
	}
}

func TestCommandAdapterExecute_NDJSON(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- ndjson-output",
				JSONFlag:       "none",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}

	dataAny, ok := res.Output["data"]
	if !ok {
		t.Fatalf("expected output[data], got keys %v", keys(res.Output))
	}
	items, ok := dataAny.([]any)
	if !ok {
		t.Fatalf("output[data] type = %T, want []any", dataAny)
	}
	if len(items) != 2 {
		t.Fatalf("len(data) = %d, want 2", len(items))
	}

	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("items[0] type = %T, want map[string]any", items[0])
	}
	if first["id"] != "abc" || first["status"] != "Up" {
		t.Fatalf("items[0] = %v, want id=abc status=Up", first)
	}

	second, ok := items[1].(map[string]any)
	if !ok {
		t.Fatalf("items[1] type = %T, want map[string]any", items[1])
	}
	if second["id"] != "def" || second["status"] != "Exited" {
		t.Fatalf("items[1] = %v, want id=def status=Exited", second)
	}

	if got := res.Output["_exit_code"]; got != float64(0) {
		t.Fatalf("output[_exit_code] = %#v, want 0", got)
	}
}

func TestCommandAdapterExecute_NDJSONMixed(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	res, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- ndjson-mixed",
				JSONFlag:       "none",
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute() error = %v, want nil", err)
	}
	if res.HTTPStatus != 200 {
		t.Fatalf("HTTPStatus = %d, want 200", res.HTTPStatus)
	}

	dataAny, ok := res.Output["data"]
	if !ok {
		t.Fatalf("expected output[data], got keys %v", keys(res.Output))
	}
	items, ok := dataAny.([]any)
	if !ok {
		t.Fatalf("output[data] type = %T, want []any", dataAny)
	}
	if len(items) != 2 {
		t.Fatalf("len(data) = %d, want 2 (noise line should be dropped)", len(items))
	}

	first, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("items[0] type = %T, want map[string]any", items[0])
	}
	if first["id"] != "abc" {
		t.Fatalf("items[0][id] = %v, want abc", first["id"])
	}
	second, ok := items[1].(map[string]any)
	if !ok {
		t.Fatalf("items[1] type = %T, want map[string]any", items[1])
	}
	if second["id"] != "def" {
		t.Fatalf("items[1][id] = %v, want def", second["id"])
	}

	if got := res.Output["_exit_code"]; got != float64(0) {
		t.Fatalf("output[_exit_code] = %#v, want 0", got)
	}
}

func TestCommandAdapterExecute_SuccessCodesStillRespectsTruncation(t *testing.T) {
	a := NewCommandAdapter(nil, 5*time.Second)

	_, err := a.Execute(context.Background(), AdapterRequest{
		Action: actions.ActionDefinition{
			Adapter: actions.AdapterConfig{
				ExecutablePath: os.Args[0],
				Command:        "-test.run=TestCommandAdapterHelperProcess -- large-success-output",
				JSONFlag:       "none",
				SuccessCodes:   []int{0, 1},
				EnvInject:      map[string]string{"GO_WANT_HELPER_PROCESS": "1"},
			},
		},
	})
	if err == nil {
		t.Fatal("expected truncation error, got nil")
	}
	if !strings.Contains(err.Error(), "command output exceeded") {
		t.Fatalf("expected truncation error, got %v", err)
	}
}

func keys(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
