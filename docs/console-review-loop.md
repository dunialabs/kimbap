# Console Review Loop (100 Rounds)

This document implements the full operating procedure for iterative review of the embedded web console served at `/console`.

Repo-specific scope:

- Console route wiring: `internal/api/routes.go`
- Embedded handler: `internal/webui/embed.go`
- Primary UI surface: `internal/webui/dist/index.html`
- Known views: `dashboard`, `actions`, `approvals`, `audit`, `services` (labeled **Vault** in navigation)

## Goal

Run 100 independent review rounds (`R001`–`R100`) with strict repetition.

Each round must include:

1. Browser inspection of all known views
2. Screenshot capture for evidence
3. Direct code fixes for confirmed issues
4. Verification (diagnostics/tests/build as relevant)
5. Regression spot-check
6. Pre-registration of the next round before closing the current one

## Per-Round SOP (must run in order)

1. **Launch app**
   - Start the server with `kimbap serve --console --port 8080`.
   - Confirm `/console` is reachable.
2. **Open `/console` via OpenChrome**
   - Inspect views in fixed order: `dashboard` → `actions` → `approvals` → `audit` → `services`.
3. **Inspect UI/UX + wording**
   - Review hierarchy, spacing, label clarity, button text, empty states, auth-gates, and error states.
4. **Capture screenshots + findings**
   - Save at least one screenshot per inspected view/state.
5. **Apply minimal direct fixes**
   - Prefer smallest safe patch, usually in `internal/webui/dist/index.html`.
6. **Run verification**
   - Recheck changed views in browser.
   - Run targeted diagnostics/tests/build for changed surface.
7. **Regression spot-check**
   - Check all changed views and at least one untouched neighboring view.
8. **Queue next round before close**
   - `R001` queues `R002`, …, `R099` queues `R100`.
   - `R100` queues terminal wrap-up record (`WRAPUP`) instead of `R101`.

## Required Evidence Per Round

- Round ID
- Launch command + URL
- `coverage.required_view_order`
- `coverage.known_views_inspected`
- Screenshot inventory
- Findings with disposition (fixed/blocked)
- Changed file list
- `wording_changes` (before/after wording updates)
- `verification.commands`
- Regression spot-check result
- Oracle review result and rerun count
- `round_result.type`
- Next-round record ID

Use: `docs/console-review-round-template.yaml`

## No-Op Round Policy

A round with no code changes is valid only when all of the following are present:

- Full view sweep evidence (all known views)
- Screenshot coverage for inspected views
- Launch + `/console` smoke proof
- Regression spot-check notes
- Next-round pre-registration

No-op rounds without evidence are invalid.

## Quality Gates

Before closing a round:

1. All known views inspected in-browser
2. All confirmed issues fixed or explicitly blocked with reason
3. Browser recheck completed for changed views
4. Verification passed for changed surface
5. Regression spot-check passed
6. Next round queued

## Suggested Commands

Initialize artifacts and round scaffolding:

```bash
make review-init
```

`make review-init` pre-registers `R001`–`R100` and creates the terminal `WRAPUP` record.

Validate a round report:

```bash
make review-validate REPORT=artifacts/console-review/R001/report.yaml
```

## Review Heuristics (applied every round)

- **Clarity**: wording is specific, concise, and actionable
- **Consistency**: same concept uses same term across all views
- **State quality**: empty/error/auth states are understandable and helpful
- **Layout coherence**: visual hierarchy and spacing support quick scanning
- **Accessibility visibility**: focus cues and contrast appear usable

## Notes

- Do not invent unavailable states. If token/data-dependent states are unreachable, mark them as blocked with evidence.
- Keep fixes minimal and targeted.
