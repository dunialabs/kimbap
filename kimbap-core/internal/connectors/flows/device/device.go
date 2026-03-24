package device

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
	"unsafe"

	"github.com/dunialabs/kimbap-core/internal/connectors"
)

type DeviceFlowConfig struct {
	DeviceEndpoint string
	TokenEndpoint  string
	ClientID       string
	ClientSecret   string
	Scopes         []string
	Timeout        time.Duration
}

type DeviceFlowResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	Scope        string
}

func RunDeviceFlow(ctx context.Context, cfg DeviceFlowConfig, output io.Writer) (*DeviceFlowResult, error) {
	if strings.TrimSpace(cfg.DeviceEndpoint) == "" {
		return nil, errors.New("device endpoint is required")
	}
	if strings.TrimSpace(cfg.TokenEndpoint) == "" {
		return nil, errors.New("token endpoint is required")
	}
	if strings.TrimSpace(cfg.ClientID) == "" {
		return nil, errors.New("client id is required")
	}
	if output == nil {
		output = io.Discard
	}

	connectorCfg := connectors.ConnectorConfig{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Scopes:       cfg.Scopes,
		DeviceURL:    cfg.DeviceEndpoint,
		TokenURL:     cfg.TokenEndpoint,
	}

	deviceResponse, err := connectors.DeviceCodeRequest(connectorCfg)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(output, "\n=== Device Authentication Required ===\n")
	_, _ = fmt.Fprintf(output, "Visit: %s\n", deviceResponse.VerificationURL)
	_, _ = fmt.Fprintf(output, "Enter code: %s\n\n", deviceResponse.UserCode)

	deviceCode, err := extractDeviceCode(deviceResponse)
	if err != nil {
		return nil, err
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}

	type pollResult struct {
		token *connectors.TokenResponse
		err   error
	}
	pollCh := make(chan pollResult, 1)

	go func() {
		token, pollErr := connectors.PollForToken(connectorCfg, deviceCode, deviceResponse.Interval, timeout)
		pollCh <- pollResult{token: token, err: pollErr}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("device flow canceled: %w", ctx.Err())
	case result := <-pollCh:
		if result.err != nil {
			return nil, result.err
		}
		return &DeviceFlowResult{
			AccessToken:  result.token.AccessToken,
			RefreshToken: result.token.RefreshToken,
			ExpiresIn:    result.token.ExpiresIn,
			Scope:        result.token.Scope,
		}, nil
	}
}

func extractDeviceCode(result *connectors.DeviceFlowResult) (string, error) {
	if result == nil {
		return "", errors.New("device flow response is nil")
	}

	value := reflect.ValueOf(result).Elem()
	field := value.FieldByName("deviceCode")
	if !field.IsValid() {
		return "", errors.New("device code not found in response")
	}

	ptr := unsafe.Pointer(field.UnsafeAddr())
	deviceCode := reflect.NewAt(field.Type(), ptr).Elem().String()
	if strings.TrimSpace(deviceCode) == "" {
		return "", errors.New("device code is empty")
	}

	return deviceCode, nil
}
