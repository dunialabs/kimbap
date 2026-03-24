package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DownstreamTransportType string

const (
	DownstreamTransportStdio DownstreamTransportType = "stdio"
	DownstreamTransportHTTP  DownstreamTransportType = "http"
	DownstreamTransportSSE   DownstreamTransportType = "sse"
)

type CreatedTransport struct {
	Transport any
	Type      DownstreamTransportType
	Cmd       *exec.Cmd
}

type DownstreamTransportFactory struct{}

func NewDownstreamTransportFactory() *DownstreamTransportFactory {
	return &DownstreamTransportFactory{}
}

func (f *DownstreamTransportFactory) DetectTransportType(launchConfig map[string]any) (DownstreamTransportType, error) {
	if t, ok := launchConfig["type"].(string); ok && t != "" {
		switch DownstreamTransportType(strings.ToLower(t)) {
		case DownstreamTransportStdio, DownstreamTransportHTTP, DownstreamTransportSSE:
			return DownstreamTransportType(strings.ToLower(t)), nil
		default:
			return "", fmt.Errorf("unsupported transport type: %s", t)
		}
	}

	if _, ok := launchConfig["command"].(string); ok {
		return DownstreamTransportStdio, nil
	}

	if rawURL, ok := launchConfig["url"].(string); ok {
		low := strings.ToLower(rawURL)
		if strings.Contains(low, "/sse") || strings.Contains(low, "/events") {
			return DownstreamTransportSSE, nil
		}
		return DownstreamTransportHTTP, nil
	}

	return "", errors.New("cannot detect transport type from launch config")
}

func (f *DownstreamTransportFactory) Create(launchConfig map[string]any) (*CreatedTransport, error) {
	typeName, err := f.DetectTransportType(launchConfig)
	if err != nil {
		return nil, err
	}

	switch typeName {
	case DownstreamTransportHTTP:
		rawURL, _ := launchConfig["url"].(string)
		if err := f.validateHTTPConfig(rawURL); err != nil {
			return nil, err
		}
		headers := headersFromLaunchConfig(launchConfig)
		var transport mcp.Transport
		streamableTransport, err := createStreamableHTTPTransport(rawURL, headers)
		if err != nil {
			transport = &mcp.SSEClientTransport{Endpoint: rawURL}
		} else {
			transport = streamableTransport
		}
		return &CreatedTransport{Transport: transport, Type: typeName}, nil
	case DownstreamTransportSSE:
		rawURL, _ := launchConfig["url"].(string)
		if err := f.validateSSEConfig(rawURL); err != nil {
			return nil, err
		}
		headers := headersFromLaunchConfig(launchConfig)
		headers["Accept"] = "text/event-stream"
		transport := &mcp.SSEClientTransport{Endpoint: rawURL, HTTPClient: httpClientWithHeaders(headers)}
		return &CreatedTransport{Transport: transport, Type: typeName}, nil
	case DownstreamTransportStdio:
		command, _ := launchConfig["command"].(string)
		if err := f.validateStdioConfig(command); err != nil {
			return nil, err
		}
		args := []string{}
		if rawArgs, ok := launchConfig["args"].([]any); ok {
			for _, arg := range rawArgs {
				if arg == nil {
					continue
				}
				args = append(args, toString(arg))
			}
		}
		cmd := exec.Command(command, args...)
		cmd.Env = mergeEnv(os.Environ(), launchConfig)
		if cwd, ok := launchConfig["cwd"].(string); ok && cwd != "" {
			cmd.Dir = cwd
		}
		return &CreatedTransport{Transport: cmd, Type: typeName, Cmd: cmd}, nil
	default:
		return nil, errors.New("unsupported transport type")
	}
}

func mergeEnv(base []string, launchConfig map[string]any) []string {
	merged := make(map[string]string, len(base))
	order := make([]string, 0, len(base))
	for _, entry := range base {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			continue
		}
		key := parts[0]
		if _, exists := merged[key]; !exists {
			order = append(order, key)
		}
		merged[key] = parts[1]
	}

	if envRaw, ok := launchConfig["env"].(map[string]any); ok {
		for key, value := range envRaw {
			if strings.TrimSpace(key) == "" {
				continue
			}
			if value == nil {
				continue
			}
			if _, exists := merged[key]; !exists {
				order = append(order, key)
			}
			merged[key] = toString(value)
		}
	}

	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, key+"="+merged[key])
	}
	return out
}

func (f *DownstreamTransportFactory) validateStdioConfig(command string) error {
	if command == "" {
		return errors.New("stdio transport requires command parameter")
	}
	if strings.Contains(command, "..") {
		return errors.New("invalid command: path traversal detected")
	}
	return nil
}

func (f *DownstreamTransportFactory) validateHTTPConfig(rawURL string) error {
	return validateURLConfig(rawURL, "HTTP")
}

func (f *DownstreamTransportFactory) validateSSEConfig(rawURL string) error {
	return validateURLConfig(rawURL, "SSE")
}

func validateURLConfig(rawURL string, transportName string) error {
	if rawURL == "" {
		return fmt.Errorf("%s transport requires URL parameter", transportName)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return errors.New("invalid URL format")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.New("invalid URL format: must use http or https scheme")
	}
	if parsed.Fragment != "" {
		return errors.New("invalid URL format: must not contain a fragment")
	}
	return nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(string(b)), "\"", ""), "\n", ""))
}

type headerInjectingRoundTripper struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *headerInjectingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if t == nil {
		return nil, errors.New("round tripper is not configured")
	}
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	cloned := req.Clone(req.Context())
	for key, value := range t.headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		cloned.Header.Set(key, value)
	}
	return base.RoundTrip(cloned)
}

func createStreamableHTTPTransport(rawURL string, headers map[string]string) (*mcp.StreamableClientTransport, error) {
	if _, err := url.Parse(rawURL); err != nil {
		return nil, err
	}
	httpClient := httpClientWithHeaders(headers)
	return &mcp.StreamableClientTransport{Endpoint: rawURL, HTTPClient: httpClient}, nil
}

func headersFromLaunchConfig(launchConfig map[string]any) map[string]string {
	out := map[string]string{
		"Content-Type": "application/json",
		"Accept":       "application/json, text/event-stream",
	}
	raw, ok := launchConfig["headers"].(map[string]any)
	if !ok {
		return out
	}
	for key, value := range raw {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if value == nil {
			continue
		}
		out[key] = toString(value)
	}
	return out
}

func httpClientWithHeaders(headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return nil
	}
	base := http.DefaultTransport
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		cloned := transport.Clone()
		if cloned.ResponseHeaderTimeout == 0 {
			cloned.ResponseHeaderTimeout = 30 * time.Second
		}
		base = cloned
	}
	return &http.Client{
		Timeout:   0,
		Transport: &headerInjectingRoundTripper{base: base, headers: headers},
	}
}
