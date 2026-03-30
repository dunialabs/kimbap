# spotify — Common Pitfalls

## Service-Level Gotchas

### Command fails with automation permission or access errors

**Likely cause**: macOS Automation permission for Spotify was not granted

**Recovery**: Grant terminal/agent automation access to Spotify in System Settings > Privacy & Security > Automation

**Severity**: high

### search-play fails with "Message not understood" or unsupported search errors

**Likely cause**: Spotify AppleScript does not support free-text search playback APIs

**Recovery**: Use spotify URI playback via spotify.play-uri or pass a spotify:track URI to search-play

**Severity**: medium

