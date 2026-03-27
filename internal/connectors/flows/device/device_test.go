package device

import (
	"context"
	"strings"
	"testing"
)

func TestRunDeviceFlow_ValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		cfg        DeviceFlowConfig
		wantErrSub string
	}{
		{
			name:       "missing device endpoint",
			cfg:        DeviceFlowConfig{TokenEndpoint: "https://example.com/token", ClientID: "client"},
			wantErrSub: "device endpoint is required",
		},
		{
			name:       "missing token endpoint",
			cfg:        DeviceFlowConfig{DeviceEndpoint: "https://example.com/device", ClientID: "client"},
			wantErrSub: "token endpoint is required",
		},
		{
			name:       "missing client id",
			cfg:        DeviceFlowConfig{DeviceEndpoint: "https://example.com/device", TokenEndpoint: "https://example.com/token"},
			wantErrSub: "client id is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunDeviceFlow(context.Background(), tc.cfg, nil)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}
