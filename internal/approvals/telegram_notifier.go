package approvals

import (
	"context"
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
func withBaseURL(baseURL string) TelegramOption {
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

	text := fmt.Sprintf("[Kimbap] Approval Required\nService: %s.%s\nAgent: %s | Risk: %s\nID: %s\n\nkimbap approve %s",
		req.Service, req.Action, req.AgentName, req.Risk, req.ID, req.ID)

	payload := map[string]string{
		"chat_id": t.chatID,
		"text":    text,
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", t.baseURL, t.token)
	return postJSONNotification(ctx, t.client, url, payload, "telegram notifier")
}
