package approvals

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type SlackNotifier struct {
	webhookURL string
	client     *http.Client
}

func NewSlackNotifier(webhookURL string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: strings.TrimSpace(webhookURL),
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *SlackNotifier) Notify(ctx context.Context, req *ApprovalRequest) error {
	if s.webhookURL == "" {
		return fmt.Errorf("slack notifier: webhook URL is required")
	}

	text := fmt.Sprintf("[Kimbap] Approval Required: %s.%s\nAgent: %s | Risk: %s | ID: %s\n\nkimbap approve %s",
		req.Service, req.Action, req.AgentName, req.Risk, req.ID, req.ID)

	return postJSONNotification(ctx, s.client, s.webhookURL, map[string]string{"text": text}, "slack notifier")
}
