package config

import (
	"strconv"
	"strings"
)

var REVERSE_REQUEST_TIMEOUTS = map[string]int{
	"sampling":    60000,
	"elicitation": 300000,
	"roots":       10000,
}

func GetReverseRequestTimeout(requestType string) int {
	key := strings.ToLower(strings.TrimSpace(requestType))
	defaultVal, ok := REVERSE_REQUEST_TIMEOUTS[key]
	if !ok {
		return 0
	}
	envKey := "REVERSE_REQUEST_TIMEOUT_" + strings.ToUpper(key)
	if envValue := strings.TrimSpace(Env(envKey)); envValue != "" {
		if parsed, err := strconv.Atoi(envValue); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultVal
}
