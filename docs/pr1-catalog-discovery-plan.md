# PR1 Catalog Discovery Plan

## Roadmap Review

The 4-main-PR roadmap is directionally correct and still matches the current codebase after pulling `origin/main`.

What should change:

1. Keep PR 1 as the starting PR.
   - It still has the cleanest review boundary.
   - It only depends on embedded catalog manifests plus existing installed-service state.
   - It avoids generator/runtime/schema risk while still adding visible capability.

2. Tighten PR 1 around a reusable read-only catalog metadata layer.
   - `service list --available`, `service search`, and `service describe` all need the same parsed catalog data.
   - Re-parsing manifests independently inside each command would duplicate logic and make later doc/test changes noisy.
   - The first implementation step should therefore be a shared catalog summary/index builder in `cmd/kimbap/`.

3. Treat read-only behavior as part of the PR contract.
   - Discovery commands should use `loadAppConfigReadOnly()` where possible.
   - They must not create `data_dir` as a side effect.
   - This is already an established CLI quality bar in `cmd/kimbap/readonly_commands_test.go`.

4. Keep PR 1 focused on catalog discovery, not init/onboarding UX.
   - `init --services select` could also benefit from richer catalog metadata, but pulling that into PR 1 widens the review surface.
   - Reuse-ready helpers are fine; changing interactive selection UX is not required in this PR.

5. Limit docs churn to surfaces affected by the new commands.
   - Required: `docs/cli-reference.md`.
   - Optional only if needed for discoverability: small `README.md` note.
   - Avoid broader docs edits in PR 1.

## PR 1 Goals

Make built-in catalog services discoverable before installation by adding catalog-native search and inspection.

User-visible outcomes:

- `kimbap service search <query>`
- `kimbap service describe <name>`
- richer `kimbap service list --available --format json`

Non-goals:

- no manifest schema changes
- no runtime adapter changes
- no generator/OpenAPI work
- no init flow redesign

## UX Contract

### `kimbap service search <query>`

Search sources:

- service name
- service description
- adapter type
- trigger verbs/objects/instead_of/exclusions
- action names
- action descriptions
- recipe names and descriptions

Ordering:

- score descending
- then name ascending for stability

Flags:

- `--limit` with the same semantics as top-level `kimbap search`

Text output:

- one row per service
- show service name, short description, install status, and a compact match summary
- end with a pointer to `kimbap service describe <name>`

JSON output:

- array of search results
- include at least: `name`, `description`, `adapter`, `auth_required`, `installed`, `enabled`, `score`, `matched_actions`, `matched_fields`

### `kimbap service describe <name>`

Lookup:

- catalog service name only
- preserve existing “did you mean” style guidance on miss

Text output should include:

- name and description
- adapter
- auth summary
- install status and shortcuts if installed
- trigger summary
- action count plus action list with descriptions
- gotcha count plus concise gotcha summary
- recipe count plus concise recipe summary
- install hint

JSON output should expose the full structured summary used by the text renderer.

### `kimbap service list --available`

Keep current text table stable enough for humans.

JSON output should be enriched with:

- `description`
- `adapter`
- `actions` as an integer count for parity with `kimbap service list`
- `auth_required`
- `triggers` as a structured object with `task_verbs`, `objects`, `instead_of`, and `exclusions`

Keep existing fields:

- `name`
- `catalog`
- `installed`
- `enabled`
- `status`
- `shortcuts`

## Data Design

Add a shared catalog helper in `cmd/kimbap/` that:

1. loads embedded catalog names
2. parses manifests once per invocation
3. merges installed-service state and shortcut aliases when config is available
4. returns stable summary structs for list/search/describe

Suggested internal structs:

- `catalogServiceSummary`
- `catalogActionSummary`
- `catalogSearchResult`

Important derived fields:

- `auth_required`: true when service-level or action-level auth is not `none`
- `triggers`: normalized string slices copied from manifest trigger metadata
- `matched_actions`: action names whose name/description matched the search query

## Implementation Steps

1. Add a shared catalog summary/index helper file under `cmd/kimbap/`.
   - Parse catalog manifests through `services.ParseManifest`.
   - Merge installed/enabled/shortcut state from the local installer.
   - Use `loadAppConfigReadOnly()` for discovery commands.

2. Refactor `newServiceListCommand()` to reuse the catalog summary helper for `--available`.
   - Preserve current text columns.
   - Extend JSON payload only.

3. Add `newServiceSearchCommand()` in a new file.
   - Reuse existing `splitSearchTerms()` pattern.
   - Implement explicit per-field scoring instead of opaque fuzzy logic.
   - Return empty JSON array or “No matching catalog services found.” consistently.

4. Add `newServiceDescribeCommand()` in a new file.
   - Reuse catalog summary helper.
   - Provide install hint plus suggestion for unknown names.

5. Register new commands from `newServiceCommand()`.

6. Update CLI docs.
   - document `service search`
   - document `service describe`
   - update `service list --available` JSON/behavior description

## Test Plan

Add targeted coverage in `cmd/kimbap/`:

1. `service list --available` JSON contains enriched fields for a known catalog service.
2. `service list --available` still reflects installed/enabled/shortcuts state.
3. `service search` finds matches by:
   - service description
   - trigger metadata
   - action name
   - action description
4. `service search --limit` truncates results deterministically.
5. `service search` returns empty result cleanly for text and JSON.
6. `service describe` returns structured JSON for a known service.
7. `service describe` text output includes install hint and action summary.
8. `service describe` unknown service returns a helpful suggestion.
9. read-only guarantee:
   - `service list --available`
   - `service search`
   - `service describe`
   must not materialize `data_dir`.

Prefer existing real catalog fixtures over synthetic manifests unless a synthetic case is required.

## Review Checklist Before Coding

- Is every new command read-only?
- Is catalog parsing centralized in one helper path?
- Does JSON output remain backward compatible except for additive fields?
- Does text output stay concise and deterministic?
- Are unknown-service errors consistent with current CLI hint style?
- Are tests covering both installed-state merge and pure catalog discovery?
- Are docs limited to directly affected surfaces?

## Review Checklist After Coding

- Run focused Go tests for `cmd/kimbap` and related package coverage touched by the change.
- Inspect JSON payloads for field names and types.
- Verify no command creates missing `data_dir`.
- Verify text output is readable without forcing README-scale formatting changes.
- Review diff for accidental scope creep into generator/runtime/init flows.
