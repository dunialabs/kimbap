# Output Filtering

Kimbap's output filter system reduces the number of tokens sent to an LLM consumer by reshaping external service responses before they reach the agent. Filters are declared directly in the service YAML manifest — no code changes required.

---

## Why it matters

External APIs are noisy. A single GitHub issue response includes `body_html`, `reactions`, `timeline_url`, `performed_via_github_app`, and dozens of URL templates the LLM never needs. Without filtering, kimbap forwards the entire payload verbatim, inflating context with data that has zero signal value.

Measured reductions using the built-in service filter configs on representative fixtures:

| Service | Original | Filtered | Reduction |
|---------|----------|----------|-----------|
| `github.list-repos` (20 repos) | 124 KB | 6.9 KB | **94%** |
| `github.get-issue` (15 issues, representative) | 50 KB | 5.7 KB | **88%** |
| `notion.query-database` (10 pages) | 21 KB | 2.4 KB | **89%** |
| `slack.get-channel-history` (20 messages) | 18 KB | 3.1 KB | **83%** |

---

## How it works

Filtering is applied **after** execution and **after** the audit record is written. The audit trail always captures the complete unfiltered response.

```
External API → Adapter → AdapterResult.Output (full)
                                ↓
                          Audit record written (full output)
                                ↓
                    ApplyFilter  →  ApplyTextFilter  →  ApplyBudget  →  ApplyCompactTemplate
                                ↓
                     ExecutionResult.Output (filtered)
                                ↓
                          LLM consumer
```

All four transformation stages are skipped when `_output_mode: raw` is passed by the caller.

The transformation stages are independent and composable:

1. **Structural filter** — select specific fields, exclude noisy fields, limit array size, strip null values.
2. **Text filter** — line-based filtering for raw command/applescript output.
3. **Budget enforcement** — hard cap on serialized output bytes; trims arrays then truncates long strings.
4. **Compact template** — render array output as a text summary using Go's `text/template` syntax.

---

## Architecture

### Adapter-agnostic design

All three adapter types (HTTP, Command, AppleScript) return `AdapterResult.Output map[string]any`. The filter operates purely on this shared type — adapters themselves are not modified. The same `FilterConfig` works identically for an HTTP REST call, a CLI subprocess, and a macOS JXA script.

```
HTTP adapter    ──┐
Command adapter ──┼──→  map[string]any  ──→  ApplyFilter  ──→  filtered map[string]any
AppleScript     ──┘
```

### Payload root detection

Real-world API responses wrap their data in inconsistent keys. `detectAndFilter` in `internal/runtime/output_filter.go` handles two cases before calling `DetectPayloadRoot`:

**Fast path — raw text output**: If the output map contains `raw` plus only runtime metadata (such as `_exit_code`), structural field selection is skipped. This covers CLI tools and AppleScript commands that return plain text instead of JSON.

**Normal path — structured JSON**: `pathutil.DetectPayloadRoot` identifies the data array by checking wrapper keys in priority order:

| Priority | Wrapper key | Source |
|---|---|---|
| 1 | `items` | HTTP pagination (`executeWithPagination`) |
| 2 | `data` | HTTP `normalizeOutput` / Command / AppleScript non-map array output |
| 3 | `result` | Legacy external response shape (recognized, not emitted by current kimbap adapters) |
| 4 | *(first array-valued key)* | Fallback for non-standard shapes |
| — | *(none)* | Flat object — filter applied directly to the map |

Filters are applied to the array items (or the single object), then the wrapper key is preserved in the output. The `_pagination` key is always passed through unchanged.

### Insertion point

Filtering is wired in `internal/runtime/pipeline.go` at the end of `executeFromCredentialsWithState`, after `finalizeWithStatus` returns:

```go
// Audit sees the full unfiltered output.
finalResult := r.finalizeWithStatus(ctx, ...)

// Success outputs can run structural filter, text filter, budget, and compact.
if finalResult.Status == actions.StatusSuccess {
    outputMode, _ := req.Input["_output_mode"].(string)
    rawMode := outputMode == "raw"

    if req.Action.FilterConfig != nil && !rawMode {
        filtered, filterMeta, _ := ApplyFilter(finalResult.Output, req.Action.FilterConfig)
        finalResult.Output = filtered
    }
    if req.Action.TextFilterConfig != nil && !rawMode {
        finalResult.Output, _ = ApplyTextFilter(finalResult.Output, req.Action.TextFilterConfig)
    }
    if !rawMode {
        if budget := coerceBudgetInt(req.Input["_budget"]); budget > 0 {
            finalResult.Output, _ = ApplyBudget(finalResult.Output, budget)
        }
    }
    if req.Action.CompactTemplate != nil && !rawMode {
        finalResult.Output, _ = ApplyCompactTemplate(finalResult.Output, req.Action.CompactTemplate)
    }
}

// Adapter error outputs keep their adapter payload and may still run text_filter.
```

### Type flow

```
manifest.FilterSpec     ──convertFilterSpec()──────→  actions.FilterConfig      ──→  ActionDefinition.FilterConfig
manifest.TextFilterSpec ──convertTextFilterSpec()──→  actions.TextFilterConfig  ──→  ActionDefinition.TextFilterConfig
manifest.CompactSpec    ──convertCompactSpec()─────→  actions.CompactTemplate   ──→  ActionDefinition.CompactTemplate
```

`internal/services/converter.go` maps manifest types to runtime types for all three adapter paths. `_output_mode` and `_budget` parameters are automatically injected into the action's input schema whenever `filter`, `text_filter`, or `compact` is present.

---

## Manifest reference

### `response.filter`

Structural field shaping applied to every response.

```yaml
response:
  filter:
    select:                          # Whitelist (output_key: source_path)
      id: id
      title: title
      state: state                   # top-level field
      assignee_login: assignee.login # nested path
    exclude:                         # Blacklist (applied after select)
      - body_html
      - reactions
    max_items: 25                    # Truncate arrays to this length
    drop_nulls: true                 # Remove null-valued object fields (recursive)
```

**Precedence**: `select` is a whitelist. If `select` is configured, only the listed fields survive. `exclude` is then applied to the remaining fields. If only `exclude` is configured without `select`, all other fields pass through.

**Select path syntax**: dot-separated for nested access, e.g. `assignee.login` extracts `response["assignee"]["login"]`. The output key (left side) is what appears in the filtered result; the source path (right side) is the extraction path in the original response.

**Array vs object**: When the payload is an array, `select`, `exclude`, `max_items`, and `drop_nulls` are applied to each item. When the payload is a single object (no array wrapper detected), `max_items` is a no-op and the other operations apply directly to the object's fields.

**Error behavior**:

| Scenario | Result |
|---|---|
| Some select paths absent in a particular item | Missing paths omitted from that item only |
| All select paths absent from every item in the array | Error; unfiltered output returned; warning in `Meta["filter_error"]` |
| Some select paths absent from every item | Partial miss; those paths recorded in `Meta["filter_partial_miss"]` |
| Raw text output (`{"raw": "..."}`) | `select` and `exclude` skipped; `drop_nulls` applies |
| Error response (non-success status) | Filter not applied |

### `response.text_filter`

Line-based filtering for raw text output from command/applescript actions.

```yaml
response:
  text_filter:
    strip_lines_matching:
      - "^Collecting"
      - "^Downloading"
    keep_lines_matching:
      - "ERROR"
      - "FAILED"
    dedup: true
    max_lines: 50
    strip_ansi: true
    on_empty: "ok"
```

Order of operations:

1. strip ANSI escape sequences (optional)
2. drop lines matching `strip_lines_matching`
3. keep only lines matching `keep_lines_matching` (if set)
4. deduplicate consecutive identical lines
5. truncate to `max_lines`
6. replace empty output with `on_empty` (if set)

`text_filter` applies to raw text success output, and also to adapter error output when the adapter returns a raw payload.

### `response.compact`

Renders array output as a human-readable text summary. Applied after `filter`.

```yaml
response:
  compact:
    header: "Issues ({{.Total}} total):"        # optional; .Total and .Count available
    item: "  #{{.number}} {{.title}} [{{.state}}]"   # required; Go text/template, all item fields available
    footer: "Showing {{.Count}} of {{.Total}}"  # optional; .Total, .Count, and .Remaining (always 0) available
```

Output shape:
```json
{
  "summary": "Issues (2 total):\n  #1 Fix crash [open]\n  #2 Add test [closed]\nShowing 2 of 2",
  "_compact": true,
  "_original_items": 2
}
```

### Reserved input parameters

When `filter` or `compact` is configured, two parameters are automatically added to the action's input schema:

| Parameter | Type | Effect |
|---|---|---|
| `_output_mode` | `"default" \| "raw"` | `"raw"` bypasses all output transformations (filter, text_filter, budget, compact) |
| `_budget` | `integer` | Maximum output size in bytes; trims arrays then truncates strings |

These do not need to be declared in the manifest.

---

## Budget enforcement

`_budget` sets a ceiling on serialized output bytes. The algorithm:

1. Trim array items from the end until output fits within the budget.
2. If the array is empty and output still exceeds the budget, progressively halve a string-truncation threshold and truncate long strings until the budget is met or the threshold drops below 10 characters.
3. Truncation is rune-aware (no splitting of multi-byte UTF-8 characters).
4. Truncated strings get a `"..."` suffix only when it actually reduces the string length.
5. If the budget cannot be met after 10 passes, the best-effort result is returned.

`_budget` is a request-level override — it is not set in the manifest. This allows different consumers to apply different budget constraints on the same action.

---

## Filter metrics

When a filter is applied, `ExecutionResult.Meta` is populated:

| Key | Type | Condition | Description |
|---|---|---|---|
| `filter_applied` | bool | always | Whether the filter ran and produced output |
| `filter_original_bytes` | int | always | Serialized size before filtering |
| `filter_result_bytes` | int | always | Serialized size after filtering |
| `filter_items_truncated_from` | int | when truncated | Original array length before `max_items` |
| `filter_partial_miss` | []string | when present | Select paths absent from every item in the array |
| `filter_skipped` | string | when skipped | Reason (e.g. `"raw_output"`) |
| `filter_error` | string | on total miss | Error when all select paths missed across all items |
| `budget_applied` | bool | when shrunken | Whether budget enforcement actually reduced the output |
| `budget_limit` | int | when shrunken | The `_budget` value |
| `budget_original_bytes` | int | when shrunken | Serialized size before budget enforcement |
| `budget_result_bytes` | int | when shrunken | Serialized size after budget enforcement |
| `compact_error` | string | on error | Template error message if compact rendering failed |

---

## Key source files

| File | Purpose |
|---|---|
| `internal/runtime/output_filter.go` | Core transformation functions: `ApplyFilter`, `ApplyTextFilter`, `ApplyBudget`, `ApplyCompactTemplate` |
| `internal/pathutil/pathutil.go` | Shared path utilities: `ExtractByPath`, `ExtractSegment`, `DetectPayloadRoot` |
| `internal/runtime/pipeline.go` | Filter insertion point after `finalizeWithStatus` |
| `internal/actions/types.go` | `FilterConfig`, `TextFilterConfig`, `FilterMeta`, `CompactTemplate` types |
| `internal/services/manifest.go` | `FilterSpec`, `TextFilterSpec`, `CompactSpec` YAML types |
| `internal/services/converter.go` | Filter/text-filter/compact conversion for all 3 adapters |
| `services/catalog/github.yaml` | Example: GitHub filter configs |
| `services/catalog/slack.yaml` | Example: Slack filter configs |
| `services/catalog/notion.yaml` | Example: Notion filter configs |
| `services/catalog/linear.yaml` | Example: Linear filter configs |
| `internal/runtime/output_filter_test.go` | Unit tests covering select, exclude, max_items, drop_nulls, budget, compact, and edge cases |

---

## Adding filters to a service

Annotate the `response` block of any action in your service manifest:

```yaml
actions:
  list-issues:
    method: GET
    path: /repos/{owner}/{repo}/issues
    args:
      owner: { type: string, required: true }
      repo:  { type: string, required: true }
    request:
      path_params:
        owner: "{owner}"
        repo:  "{repo}"
    response:
      type: array
      filter:
        select:
          id:         id
          number:     number
          title:      title
          state:      state
          html_url:   html_url
          user_login: user.login
        max_items: 25
        drop_nulls: true
```

Validate before installing:

```bash
kimbap service validate my-service.yaml
```

Output transformation configs are validated at manifest load time:
- `max_items` must be ≥ 0.
- `select` keys and source paths must be non-empty strings.
- `text_filter.max_lines` must be ≥ 0.
- `text_filter` regex lists must contain non-empty valid regex patterns.
- `compact.item` is required when `compact` is present.
