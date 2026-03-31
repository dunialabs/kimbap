package headerutil

import "testing"

func TestIsCredentialLikeHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{name: "authorization", header: "Authorization", want: true},
		{name: "api key exact", header: "X-API-Key", want: true},
		{name: "api key split words", header: "X-Api_Key", want: true},
		{name: "token part", header: "X-Session-Token", want: true},
		{name: "secret part", header: "X-Client-Secret", want: true},
		{name: "normal header", header: "Content-Type", want: false},
		{name: "substring only", header: "X-Tokenized-Response", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCredentialLikeHeader(tc.header); got != tc.want {
				t.Fatalf("IsCredentialLikeHeader(%q) = %v, want %v", tc.header, got, tc.want)
			}
		})
	}
}
