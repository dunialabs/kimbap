package adapters

import (
	"context"
	"net/http"
	"strings"

	"github.com/dunialabs/kimbap/internal/actions"
)

const defaultHackerNewsFeedLimit = 10
const maxHackerNewsFeedLimit = 50

func (a *HTTPAdapter) postProcessOutput(ctx context.Context, req AdapterRequest, output map[string]any) (map[string]any, *actions.ExecutionError) {
	switch strings.TrimSpace(req.Action.Name) {
	case "hacker-news.get-top-stories", "hacker-news.get-new-stories", "hacker-news.get-best-stories":
		return a.hydrateHackerNewsFeed(ctx, req, output)
	default:
		return output, nil
	}
}

func (a *HTTPAdapter) hydrateHackerNewsFeed(ctx context.Context, req AdapterRequest, output map[string]any) (map[string]any, *actions.ExecutionError) {
	ids, ok := extractResponseItems(output)
	if !ok || len(ids) == 0 {
		return output, nil
	}

	limit := hackerNewsFeedLimit(req.Input)
	stories := make([]any, 0, min(limit, len(ids)))
	for _, rawID := range ids {
		itemID, ok := hackerNewsItemID(rawID)
		if !ok {
			continue
		}
		item, execErr := a.fetchHackerNewsItem(ctx, req, itemID)
		if execErr != nil {
			return nil, execErr
		}
		if item == nil {
			continue
		}
		stories = append(stories, item)
		if len(stories) >= limit {
			break
		}
	}

	return map[string]any{"data": stories}, nil
}

func hackerNewsFeedLimit(input map[string]any) int {
	if n, ok := positiveIntFromAny(input["limit"]); ok {
		if n > maxHackerNewsFeedLimit {
			return maxHackerNewsFeedLimit
		}
		return n
	}
	return defaultHackerNewsFeedLimit
}

func hackerNewsItemID(value any) (int, bool) {
	return positiveIntFromAny(value)
}

func (a *HTTPAdapter) fetchHackerNewsItem(ctx context.Context, req AdapterRequest, itemID int) (map[string]any, *actions.ExecutionError) {
	itemReq := req
	itemReq.Action = actions.ActionDefinition{
		Name: req.Action.Namespace + ".get-item",
		Adapter: actions.AdapterConfig{
			Type:          "http",
			Method:        http.MethodGet,
			BaseURL:       req.Action.Adapter.BaseURL,
			URLTemplate:   "/item/{id}.json",
			AllowInsecure: req.Action.Adapter.AllowInsecure,
			Retry:         req.Action.Adapter.Retry,
		},
		Auth: req.Action.Auth,
	}
	itemReq.Input = map[string]any{"id": itemID}

	result, err := a.executeSingle(ctx, itemReq)
	if err != nil {
		if execErr, ok := err.(*actions.ExecutionError); ok {
			return nil, execErr
		}
		return nil, actions.NewExecutionError(actions.ErrDownstreamUnavailable, err.Error(), http.StatusBadGateway, true, nil)
	}
	return normalizeHackerNewsItem(result.Output), nil
}

func normalizeHackerNewsItem(output map[string]any) map[string]any {
	if output == nil {
		return nil
	}
	if data, exists := output["data"]; exists {
		switch typed := data.(type) {
		case nil:
			return nil
		case map[string]any:
			output = typed
		default:
			return nil
		}
	}
	if len(output) == 0 {
		return nil
	}
	if deleted, _ := output["deleted"].(bool); deleted {
		return nil
	}
	if dead, _ := output["dead"].(bool); dead {
		return nil
	}
	if itemType, _ := output["type"].(string); strings.TrimSpace(itemType) != "" && !strings.EqualFold(strings.TrimSpace(itemType), "story") {
		return nil
	}
	return output
}
