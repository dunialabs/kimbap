package device

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dunialabs/kimbap-core/internal/connectors"
)

type DeviceFlowConfig struct {
	DeviceEndpoint string
	TokenEndpoint  string
	ClientID       string
	ClientSecret   string
	Scopes         []string
	Timeout        time.Duration
	AuthMethod     string
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
		AuthMethod:   cfg.AuthMethod,
		Scopes:       cfg.Scopes,
		DeviceURL:    cfg.DeviceEndpoint,
		TokenURL:     cfg.TokenEndpoint,
	}

	deviceResponse, err := connectors.DeviceCodeRequest(connectorCfg)
	if err != nil {
		return nil, err
	}

	_, _ = fmt.Fprintf(output, "\nOpen this URL in any browser:\n  %s\n\n", deviceResponse.VerificationURL)
	_, _ = fmt.Fprintf(output, "Enter code:\n  %s\n\n", deviceResponse.UserCode)
	_, _ = fmt.Fprintf(output, "Waiting for approval... Press Ctrl+C to cancel.\n")

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
		token, pollErr := connectors.PollForTokenWithContext(ctx, connectorCfg, deviceResponse.DeviceCode, deviceResponse.Interval, timeout)
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
