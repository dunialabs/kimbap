package services

import "testing"

func TestIsValidServiceAlias(t *testing.T) {
	tests := []struct {
		name  string
		alias string
		want  bool
	}{
		{name: "valid simple", alias: "geo", want: true},
		{name: "valid hyphen", alias: "weather-geo", want: true},
		{name: "reject numeric start", alias: "1", want: false},
		{name: "reject dot", alias: "geo.search", want: false},
		{name: "reject reserved", alias: "kimbap", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidServiceAlias(tc.alias); got != tc.want {
				t.Fatalf("IsValidServiceAlias(%q) = %v, want %v", tc.alias, got, tc.want)
			}
		})
	}
}

func TestSuggestedServiceAliases_DefaultCandidates(t *testing.T) {
	got := SuggestedServiceAliases("open-meteo-geocoding", nil)
	if len(got) == 0 {
		t.Fatal("expected default alias candidates")
	}
	if got[0] != "geo" {
		t.Fatalf("expected first default alias to be geo, got %q (all=%v)", got[0], got)
	}
}

func TestSuggestedServiceAliases_PreferManifestAliases(t *testing.T) {
	got := SuggestedServiceAliases("open-meteo-geocoding", []string{"weather-geo", "geo"})
	if len(got) == 0 {
		t.Fatal("expected alias candidates")
	}
	if got[0] != "weather-geo" {
		t.Fatalf("expected manifest alias to be first, got %q (all=%v)", got[0], got)
	}
}
