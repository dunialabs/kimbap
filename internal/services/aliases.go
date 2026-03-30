package services

import "strings"

func IsValidServiceAlias(alias string) bool {
	c := strings.ToLower(strings.TrimSpace(alias))
	if c == "" || strings.Contains(c, ".") {
		return false
	}
	if !serviceNamePattern.MatchString(c) {
		return false
	}
	if err := ValidateServiceName(c); err != nil {
		return false
	}
	return true
}

func SuggestedServiceAliases(name string, preferred []string) []string {
	baseName := strings.ToLower(strings.TrimSpace(name))
	tokens := splitServiceNameTokens(baseName)
	if len(tokens) == 0 && baseName != "" {
		tokens = []string{baseName}
	}

	seen := make(map[string]struct{})
	out := make([]string, 0)
	add := func(candidate string) {
		c := strings.ToLower(strings.TrimSpace(candidate))
		if c == baseName {
			return
		}
		if !IsValidServiceAlias(c) {
			return
		}
		if _, exists := seen[c]; exists {
			return
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}

	for _, alias := range preferred {
		add(alias)
	}

	if len(tokens) > 0 {
		last := tokens[len(tokens)-1]
		if len(last) >= 3 {
			add(last[:3])
		}
		if len(last) >= 4 {
			add(last[:4])
		}
		add(last)
	}

	if len(tokens) >= 2 {
		var initials strings.Builder
		for _, tok := range tokens {
			if tok == "" {
				continue
			}
			initials.WriteByte(tok[0])
		}
		add(initials.String())
	}

	if len(tokens) > 0 {
		first := tokens[0]
		if len(first) >= 3 {
			add(first[:3])
		}
		add(first)
	}

	compact := strings.Join(tokens, "")
	if len(compact) >= 3 {
		add(compact[:3])
	}
	if len(compact) >= 4 {
		add(compact[:4])
	}

	return out
}

func PreferredServiceAlias(name string, preferred []string) string {
	candidates := SuggestedServiceAliases(name, preferred)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func splitServiceNameTokens(name string) []string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(name)), func(r rune) bool {
		return r == '-' || r == '_'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
