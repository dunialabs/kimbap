package approvals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type TelegramNotifier struct {
	token   string
	chatID  string
	baseURL string
	client  *http.Client
}

// TelegramOption is a functional option for TelegramNotifier.
type TelegramOption func(*TelegramNotifier)

// WithBaseURL overrides the Telegram API base URL. Intended for testing.
func WithBaseURL(baseURL string) TelegramOption {
	return func(t *TelegramNotifier) {
		t.baseURL = strings.TrimRight(baseURL, "/")
	}
}

func NewTelegramNotifier(token, chatID string, opts ...TelegramOption) *TelegramNotifier {
	n := &TelegramNotifier{
		token:   strings.TrimSpace(token),
		chatID:  strings.TrimSpace(chatID),
		baseURL: "https://api.telegram.org",
		client:  &http.Client{Timeout: 10 * time.Second},
	}
	for _, opt := range opts {
		opt(n)
	}
	return n
}

func (t *TelegramNotifier) Notify(ctx context.Context, req *ApprovalRequest) error {
	if t.token == "" {
		return fmt.Errorf("telegram notifier: bot token is required")
	}
	if t.chatID == "" {
		return fmt.Errorf("telegram notifier: chat ID is required")
	}

	text := fmt.Sprintf("*[Kimbap] Approval Required*\nService: `%s.%s`\nAgent: `%s` | Risk: `%s`\nID: `%s`\n\n`kimbap approve %s`",
		req.Service, req.Action, req.AgentName, req.Risk, req.ID, req.ID)

	payload, err := json.Marshal(map[string]string{
		"chat_id":    t.chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})
	if err != nil {
		return fmt.Errorf("telegram notifier: marshal payload: %w", err)
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.token)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("telegram notifier: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := t.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("telegram notifier: send: %w", err)
	}
	_ = res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("telegram notifier: unexpected status %d", res.StatusCode)
	}
	return nil
}
