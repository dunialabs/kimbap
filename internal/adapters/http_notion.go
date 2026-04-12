package adapters

import (
	"fmt"
	"strings"
)

func normalizeActionInput(actionName string, input map[string]any) (map[string]any, error) {
	normalized := cloneAnyMap(input)
	if normalized == nil {
		normalized = map[string]any{}
	}
	if strings.TrimSpace(actionName) == "notion.create-page" {
		return normalizeNotionCreatePageInput(normalized)
	}
	if strings.TrimSpace(actionName) == "notion.update-page" {
		return normalizeNotionUpdatePageInput(normalized)
	}
	return normalized, nil
}

func normalizeNotionCreatePageInput(input map[string]any) (map[string]any, error) {
	if _, ok := input["parent"]; !ok {
		if databaseID, ok := nonEmptyStringValue(input, "database_id"); ok {
			input["parent"] = map[string]any{"database_id": databaseID}
		} else if parentID, ok := nonEmptyStringValue(input, "parent_id"); ok {
			input["parent"] = map[string]any{"page_id": parentID}
		}
	}
	if _, ok := input["parent"]; !ok {
		return nil, fmt.Errorf("notion.create-page requires parent, parent_id, or database_id")
	}

	if _, ok := input["properties"]; !ok {
		if title, ok := nonEmptyStringValue(input, "title"); ok {
			titleProperty, _ := nonEmptyStringValue(input, "title_property")
			input["properties"] = notionTitleProperties(titleProperty, title)
		}
	}

	if _, ok := input["children"]; !ok {
		if content, ok := nonEmptyStringValue(input, "content"); ok {
			input["children"] = notionParagraphChildren(content)
		}
	} else if _, ok := nonEmptyStringValue(input, "content"); ok {
		return nil, fmt.Errorf("notion.create-page content cannot be combined with children")
	}

	return input, nil
}

func normalizeNotionUpdatePageInput(input map[string]any) (map[string]any, error) {
	if _, ok := input["properties"]; !ok {
		if title, ok := nonEmptyStringValue(input, "title"); ok {
			titleProperty, _ := nonEmptyStringValue(input, "title_property")
			input["properties"] = notionTitleProperties(titleProperty, title)
		}
	}
	return input, nil
}

func notionTitleProperties(titleProperty string, title string) map[string]any {
	key := strings.TrimSpace(titleProperty)
	if key == "" {
		key = "title"
	}
	return map[string]any{
		key: map[string]any{
			"title": []any{
				map[string]any{
					"text": map[string]any{"content": title},
				},
			},
		},
	}
}

func notionParagraphChildren(content string) []any {
	return []any{
		map[string]any{
			"object": "block",
			"type":   "paragraph",
			"paragraph": map[string]any{
				"rich_text": []any{
					map[string]any{
						"type": "text",
						"text": map[string]any{"content": content},
					},
				},
			},
		},
	}
}

func nonEmptyStringValue(input map[string]any, key string) (string, bool) {
	value, ok := input[key]
	if !ok || value == nil {
		return "", false
	}
	trimmed := strings.TrimSpace(toString(value))
	if trimmed == "" || trimmed == "<nil>" {
		return "", false
	}
	return trimmed, true
}
