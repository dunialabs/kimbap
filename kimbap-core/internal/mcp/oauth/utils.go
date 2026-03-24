package oauth

import "time"

const maxExpiresInSeconds int64 = 365 * 24 * 60 * 60

func ResolveExpires(responseExpiresIn *int64, defaultExpiresIn *int64) (*int64, *int64) {
	var expiresIn *int64
	if responseExpiresIn != nil {
		expiresIn = responseExpiresIn
	} else if defaultExpiresIn != nil {
		expiresIn = defaultExpiresIn
	}

	if expiresIn == nil {
		return nil, nil
	}

	clamped := *expiresIn
	if clamped < 0 {
		clamped = 0
	} else if clamped > maxExpiresInSeconds {
		clamped = maxExpiresInSeconds
	}
	clampedExpiresIn := clamped
	expiresAt := time.Now().UnixMilli() + (clamped * 1000)
	return &clampedExpiresIn, &expiresAt
}

func NumberToInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case float32:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}
