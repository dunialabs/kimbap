package approvals

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type Notifier interface {
	Notify(ctx context.Context, req *ApprovalRequest) error
}

type WebhookNotifier struct {
	url     string
	client  *http.Client
	signKey []byte
}

func NewWebhookNotifier(url string, signKey []byte) *WebhookNotifier {
	return &WebhookNotifier{
		url:     strings.TrimSpace(url),
		client:  &http.Client{Timeout: 10 * time.Second},
		signKey: append([]byte(nil), signKey...),
	}
}

const (
	webhookMaxRetries    = 3
	webhookRetryBaseWait = 500 * time.Millisecond
)

func (w *WebhookNotifier) Notify(ctx context.Context, req *ApprovalRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal approval webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= webhookMaxRetries; attempt++ {
		if attempt > 0 {
			wait := webhookRetryBaseWait * time.Duration(1<<(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(wait):
			}
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, strings.NewReader(string(body)))
		if err != nil {
			return fmt.Errorf("create approval webhook request: %w", err)
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("X-Kimbap-Signature", w.sign(body))

		res, err := w.client.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("post approval webhook: %w", err)
			continue
		}
		_ = res.Body.Close()

		if res.StatusCode >= http.StatusOK && res.StatusCode < http.StatusMultipleChoices {
			return nil
		}

		lastErr = fmt.Errorf("approval webhook returned status %d", res.StatusCode)
		if res.StatusCode < 500 {
			return lastErr
		}
	}

	return lastErr
}

func (w *WebhookNotifier) sign(body []byte) string {
	if len(w.signKey) == 0 {
		return ""
	}
	h := hmac.New(sha256.New, w.signKey)
	_, _ = h.Write(body)
	return "sha256=" + hex.EncodeToString(h.Sum(nil))
}

type LogNotifier struct{}

func (l *LogNotifier) Notify(_ context.Context, req *ApprovalRequest) error {
	log.Printf("approval pending id=%s tenant=%s service=%s action=%s", req.ID, req.TenantID, req.Service, req.Action)
	return nil
}

type MultiNotifier struct{ notifiers []Notifier }

func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	filtered := make([]Notifier, 0, len(notifiers))
	for _, notifier := range notifiers {
		if notifier != nil {
			filtered = append(filtered, notifier)
		}
	}
	return &MultiNotifier{notifiers: filtered}
}

func (m *MultiNotifier) Notify(ctx context.Context, req *ApprovalRequest) error {
	for _, notifier := range m.notifiers {
		if err := notifier.Notify(ctx, req); err != nil {
			log.Printf("approval notifier: adapter error (continuing): %v", err)
		}
	}
	return nil
}
