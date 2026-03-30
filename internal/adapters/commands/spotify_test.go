package commands

import (
	"strings"
	"testing"
)

func TestSpotifyCommandNames(t *testing.T) {
	cmds := SpotifyCommands()
	expected := []string{
		"spotify-get-current-track",
		"spotify-play",
		"spotify-pause",
		"spotify-next-track",
		"spotify-previous-track",
		"spotify-set-volume",
		"spotify-get-player-state",
		"spotify-play-uri",
		"spotify-search-play",
	}
	for _, name := range expected {
		if _, ok := cmds[name]; !ok {
			t.Fatalf("missing command %q", name)
		}
	}
}

func TestSpotifySearchPlayDoesNotUseUnsupportedSearchForCall(t *testing.T) {
	cmd := SpotifyCommands()["spotify-search-play"]
	if strings.Contains(cmd.Script, "searchFor(") {
		t.Fatalf("spotify-search-play must not call unsupported Spotify searchFor() API")
	}
	if !strings.Contains(cmd.Script, "[NOT_SUPPORTED]") {
		t.Fatalf("spotify-search-play should emit [NOT_SUPPORTED] guidance for free-text queries")
	}
	if !strings.Contains(cmd.Script, "playTrack(") {
		t.Fatalf("spotify-search-play should use playTrack() for Spotify URI playback")
	}
}

func TestSpotifyPlayURIRequiresSpotifyScheme(t *testing.T) {
	cmd := SpotifyCommands()["spotify-play-uri"]
	if !strings.Contains(cmd.Script, "uri.indexOf(\"spotify:\") !== 0") {
		t.Fatalf("spotify-play-uri should validate spotify: URI scheme")
	}
	if !strings.Contains(cmd.Script, "playTrack(uri)") {
		t.Fatalf("spotify-play-uri should call playTrack(uri)")
	}
}
