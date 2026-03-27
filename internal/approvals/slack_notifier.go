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

	payload, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("slack notifier: marshal payload: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("slack notifier: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("slack notifier: send: %w", err)
	}
	_ = res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("slack notifier: unexpected status %d", res.StatusCode)
	}
	return nil
}
