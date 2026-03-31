package approvals

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

func postJSONNotification(ctx context.Context, client *http.Client, targetURL string, payload any, errorPrefix string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("%s: marshal payload: %w", errorPrefix, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%s: create request: %w", errorPrefix, err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("%s: send: %w", errorPrefix, err)
	}
	_ = res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("%s: unexpected status %d", errorPrefix, res.StatusCode)
	}
	return nil
}
