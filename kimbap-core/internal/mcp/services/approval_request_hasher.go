package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type ArgsNormalizer func(args map[string]any) map[string]any

type ApprovalRequestHasher struct {
	normalizers map[string]ArgsNormalizer
}

func NewApprovalRequestHasher() *ApprovalRequestHasher {
	return &ApprovalRequestHasher{normalizers: map[string]ArgsNormalizer{}}
}

var approvalRequestHasher = NewApprovalRequestHasher()

func ApprovalRequestHasherInstance() *ApprovalRequestHasher { return approvalRequestHasher }

func (h *ApprovalRequestHasher) RegisterNormalizer(toolName string, normalizer ArgsNormalizer) {
	if h == nil || normalizer == nil {
		return
	}
	h.normalizers[toolName] = normalizer
}

func (h *ApprovalRequestHasher) CanonicalizeArgs(toolName string, args map[string]any) string {
	if h == nil {
		h = ApprovalRequestHasherInstance()
	}
	normalized := args
	if normalized == nil {
		normalized = map[string]any{}
	}
	if normalizer, ok := h.normalizers[toolName]; ok && normalizer != nil {
		normalized = normalizer(normalized)
	}
	cleaned := h.stripVolatileFields(normalized)
	b, err := canonicalJSON(cleaned)
	if err != nil {
		return "{}"
	}
	return b
}

func (h *ApprovalRequestHasher) ComputeHash(userID, serverID, toolName string, args map[string]any, policyVersion int) string {
	canonicalArgs := h.CanonicalizeArgs(toolName, args)
	hashInput := fmt.Sprintf("%s|%s|%s|%s|%d", userID, serverID, toolName, canonicalArgs, policyVersion)
	sum := sha256.Sum256([]byte(hashInput))
	return fmt.Sprintf("%x", sum)
}

func (h *ApprovalRequestHasher) stripVolatileFields(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case map[string]any:
		out := map[string]any{}
		for key, child := range typed {
			if key == "_meta" {
				continue
			}
			cleaned := h.stripVolatileFields(child)
			if cleaned == nil {
				continue
			}
			switch nested := cleaned.(type) {
			case []any:
				if len(nested) == 0 {
					continue
				}
			case map[string]any:
				if len(nested) == 0 {
					continue
				}
			}
			out[key] = cleaned
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			cleaned := h.stripVolatileFields(item)
			if cleaned == nil {
				continue
			}
			out = append(out, cleaned)
		}
		return out
	default:
		return typed
	}
}

func canonicalJSON(v any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	result := buf.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}
