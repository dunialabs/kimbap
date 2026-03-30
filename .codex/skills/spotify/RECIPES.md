# spotify — Recipes

## Resume and inspect current playback

Start playback and confirm current track metadata

### Steps

1. Run spotify.play to resume playback
2. Run spotify.get-current-track to confirm track name and artist

## Play an exact track URI

Start playback using a Spotify URI for deterministic results

### Steps

1. Run spotify.play-uri with spotify:track:<id>
2. Run spotify.get-player-state to verify playback is active

