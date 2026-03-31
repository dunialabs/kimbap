package headerutil

import "strings"

func IsCredentialLikeHeader(key string) bool {
	name := strings.ToLower(strings.TrimSpace(key))
	switch name {
	case "authorization", "proxy-authorization", "x-api-key", "x-auth-token", "x-access-token":
		return true
	}
	parts := headerNameParts(name)
	for i, part := range parts {
		switch part {
		case "apikey", "token", "secret":
			return true
		case "api":
			if i+1 < len(parts) && parts[i+1] == "key" {
				return true
			}
		}
	}
	return false
}

func headerNameParts(name string) []string {
	return strings.FieldsFunc(name, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}

func IsSensitiveAuditHeader(key string) bool {
	if IsCredentialLikeHeader(key) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "cookie", "set-cookie":
		return true
	default:
		return false
	}
}
