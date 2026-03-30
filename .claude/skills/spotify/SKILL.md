---
name: spotify
description: |
  Use when you need to play, pause, skip, inspect, search, control music, track, playback, volume through approved Kimbap actions.
  Use instead of: manually controlling Spotify UI for routine playback actions.
  Do not use for: non-Spotify audio players.
allowed-tools: Bash
---

# spotify

Spotify playback control via AppleScript (macOS only)

## Prerequisites

- Kimbap CLI installed and in PATH
- Service installed: `kimbap service install spotify`
- macOS app installed and automatable: `Spotify`

## Available Actions

Pick an action from this table, then run `kimbap call spotify.<action> ...`.

| Action | Description | Inputs | Risk |
|--------|-------------|--------|------|
| `spotify.get-current-track` | Get metadata for the current track and playback position | - | low |
| `spotify.get-player-state` | Get player state, current volume, and playback position | - | low |
| `spotify.next-track` | Skip to the next track | - | medium |
| `spotify.pause` | Pause Spotify playback | - | low |
| `spotify.play` | Start or resume Spotify playback | - | low |
| `spotify.play-uri` | Start playback for an exact Spotify URI (spotify:track:..., spotify:playlist:...) | `uri` | low |
| `spotify.previous-track` | Return to the previous track | - | medium |
| `spotify.search-play` | Start playback from a Spotify URI passed via query (legacy compatibility) | `query` | medium |
| `spotify.set-volume` | Set Spotify player volume from 0 to 100 | `volume` | medium |

## Top Gotchas

- **Command fails with automation permission or access errors** → Grant terminal/agent automation access to Spotify in System Settings > Privacy & Security > Automation
- **search-play fails with "Message not understood" or unsupported search errors** → Use spotify URI playback via spotify.play-uri or pass a spotify:track URI to search-play

## Files in This Pack

- **GOTCHAS.md** — Common pitfalls, error patterns, and recovery steps
- **RECIPES.md** — Multi-step workflow playbooks

## Example

```bash
kimbap call spotify.get-current-track
kimbap call spotify.get-player-state
```

## Shortcut Command Alias (optional)

```bash
kimbap alias set <shortcut> spotify.get-current-track
<shortcut>
```

## Before Execute

- Inspect: `kimbap actions describe spotify.<action> --format json`
- Preview non-low-risk actions: `kimbap call --format json spotify.<action> --dry-run`
- Read GOTCHAS.md in this pack before unfamiliar or risky actions
- Read RECIPES.md in this pack for workflow examples
