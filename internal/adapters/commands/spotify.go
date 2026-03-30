package commands

func SpotifyCommands() map[string]Command {
	return map[string]Command{
		"spotify-get-current-track": {
			Name: "spotify-get-current-track", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

var track = app.currentTrack();
if (!track) throw new Error("[NOT_FOUND] no current track");

var result = {
	name: track.name(),
	artist: track.artist(),
	album: track.album(),
	duration_ms: track.duration(),
	position_seconds: app.playerPosition()
};
JSON.stringify(result);`,
		},
		"spotify-play": {
			Name: "spotify-play", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

app.play();
var result = {
	state: String(app.playerState()),
	position_seconds: app.playerPosition()
};
JSON.stringify(result);`,
		},
		"spotify-pause": {
			Name: "spotify-pause", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

app.pause();
var result = {
	state: String(app.playerState()),
	position_seconds: app.playerPosition()
};
JSON.stringify(result);`,
		},
		"spotify-next-track": {
			Name: "spotify-next-track", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

app.nextTrack();
var track = app.currentTrack();
if (!track) throw new Error("[NOT_FOUND] no current track");

var result = {
	name: track.name(),
	artist: track.artist(),
	album: track.album(),
	position_seconds: app.playerPosition()
};
JSON.stringify(result);`,
		},
		"spotify-previous-track": {
			Name: "spotify-previous-track", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

app.previousTrack();
var track = app.currentTrack();
if (!track) throw new Error("[NOT_FOUND] no current track");

var result = {
	name: track.name(),
	artist: track.artist(),
	album: track.album(),
	position_seconds: app.playerPosition()
};
JSON.stringify(result);`,
		},
		"spotify-set-volume": {
			Name: "spotify-set-volume", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

if (typeof input.volume !== "number") throw new Error("volume is required");
if (input.volume < 0 || input.volume > 100) throw new Error("volume must be between 0 and 100");

var volume = Math.round(input.volume);
app.setSoundVolume(volume);

var result = {
	volume: app.soundVolume()
};
JSON.stringify(result);`,
		},
		"spotify-get-player-state": {
			Name: "spotify-get-player-state", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

var result = {
	state: String(app.playerState()),
	volume: app.soundVolume(),
	position_seconds: app.playerPosition()
};
JSON.stringify(result);`,
		},
		"spotify-play-uri": {
			Name: "spotify-play-uri", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

if (!input.uri) throw new Error("uri is required");
var uri = String(input.uri).trim();
if (!uri) throw new Error("uri is required");
if (uri.indexOf("spotify:") !== 0) throw new Error("[NOT_SUPPORTED] uri must start with spotify:");

try {
	app.playTrack(uri);
} catch (e) {
	throw new Error("[NOT_FOUND] spotify uri not playable");
}

var result = {
	uri: uri,
	state: String(app.playerState())
};
JSON.stringify(result);`,
		},
		"spotify-search-play": {
			Name: "spotify-search-play", TargetApp: "Spotify",
			Script: stdinReader + `
var app = Application("Spotify");
app.includeStandardAdditions = false;

if (!input.query) throw new Error("query is required");

var query = String(input.query).trim();
if (!query) throw new Error("query is required");

if (query.indexOf("spotify:") !== 0) {
	throw new Error("[NOT_SUPPORTED] free-text Spotify search is not available via AppleScript. Use spotify URI (spotify:track:...) with query or use spotify-play-uri.");
}

try {
	app.playTrack(query);
} catch (e) {
	throw new Error("[NOT_FOUND] spotify uri not playable");
}

var result = {
	query: query,
	state: String(app.playerState())
};
JSON.stringify(result);`,
		},
	}
}
