package core

import (
	"encoding/json"
	"strconv"
	"strings"

	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
)

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "null"
	}
	return string(b)
}

func extractStreamID(eventID mcptypes.EventID) mcptypes.StreamID {
	raw := string(eventID)
	lastUnderscore := strings.LastIndex(raw, "_")
	if lastUnderscore == -1 {
		return mcptypes.StreamID(raw)
	}
	secondLastUnderscore := strings.LastIndex(raw[:lastUnderscore], "_")
	if secondLastUnderscore == -1 {
		return mcptypes.StreamID(raw)
	}
	return mcptypes.StreamID(raw[:secondLastUnderscore])
}

func extractTimestampFromEventID(eventID mcptypes.EventID) int64 {
	raw := string(eventID)
	lastUnderscore := strings.LastIndex(raw, "_")
	if lastUnderscore == -1 {
		return 0
	}
	secondLastUnderscore := strings.LastIndex(raw[:lastUnderscore], "_")
	if secondLastUnderscore == -1 {
		return 0
	}
	ts, err := strconv.ParseInt(raw[secondLastUnderscore+1:lastUnderscore], 10, 64)
	if err != nil {
		return 0
	}
	return ts
}
