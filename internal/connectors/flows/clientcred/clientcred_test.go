package clientcred

import (
	"context"
	"strings"
	"testing"
)

func TestRunClientCredentialsFlow_ValidatesRequiredFields(t *testing.T) {
	tests := []struct {
		name       string
		cfg        ClientCredentialsConfig
		wantErrSub string
	}{
		{
			name:       "missing token endpoint",
			cfg:        ClientCredentialsConfig{ClientID: "client"},
			wantErrSub: "token endpoint is required",
		},
		{
			name:       "missing client id",
			cfg:        ClientCredentialsConfig{TokenEndpoint: "https://example.com/token"},
			wantErrSub: "client id is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := RunClientCredentialsFlow(context.Background(), tc.cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tc.wantErrSub) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErrSub)
			}
		})
	}
}
