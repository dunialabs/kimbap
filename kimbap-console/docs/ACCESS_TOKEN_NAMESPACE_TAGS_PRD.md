# PRD: Access Token Namespaces & Tags (Console-Only)

## Summary

KIMBAP Console access tokens currently have **name/notes/permissions**, but lack a first-class way to **group** tokens and apply **bulk changes**. This PRD introduces:

- **Namespace** (single string) + **Tags** (string list) as token metadata
- **Bulk update** from Console (UI + internal `/api/v1` protocol + external REST API)
- **Console-only implementation**: KIMBAP Core is **not modified**; Console stores metadata and fans out updates to Core Admin APIs.

## Goals

- Allow admins to assign and edit `namespace` + `tags` on access tokens.
- Allow bulk operations for multiple tokens:
  - Apply a permissions set to many tokens at once
  - Update `namespace` for many tokens
  - Add/remove/replace/clear `tags` for many tokens
- Support both:
  - Console UI workflows
  - External REST API workflows (batch create already exists; batch update added)

## Non-Goals

- No changes to **KIMBAP Core** schema/APIs.
- No server-side enforcement in Core based on `namespace/tags` (metadata is Console-owned).
- No “permission templates” as a separate saved entity (bulk permission apply uses an existing token as a source).

## Concepts & Terminology

- **Access Token**: Core “user” record (role 2=admin, 3=member). Console stores the plaintext token in its local `user` table for auth and proxy calls.
- **Namespace (Workspace)**: A single grouping key (e.g., `default`, `team-a`, `prod`). In Console UX this can be presented as a “workspace” grouping concept without requiring Core changes.
- **Tags**: Multiple labels per token (e.g., `["ci","prod"]`).
- **Bulk Update**: Console receives a request with a token list and applies updates to Core (permissions) and Console DB (metadata) per token.

## Authorization & RBAC

- **External REST API** (`/api/external/*`) requires an **Owner token** (role=1). This is enforced by `app/api/external/lib/auth.ts`.
- **Console UI**:
  - Owner/Admin can **create/edit/delete** tokens and use **bulk update**.
  - Member is **read-only** on the Access Token Management page (no create/edit/delete/bulk update actions shown).
- **Server-side enforcement**:
  - Core Admin APIs are the ultimate authority for permission changes.
  - Console-owned metadata updates should follow the same intent: only Owner/Admin should be able to modify metadata.

## Data Model (Console DB)

Create a Console-only table for metadata:

- Table: `token_metadata`
- Primary key: `(proxy_id, userid)`
- Columns:
  - `proxy_id` (int)
  - `userid` (varchar) – Core userId/tokenId
  - `namespace` (varchar, default `default`)
  - `tags` (text JSON, default `[]`)
  - `created_at`, `updated_at`

Rationale:
- Avoid Core changes while still enabling filtering/grouping and bulk update targeting in Console.

## Backward Compatibility & Defaults

- Existing tokens without a metadata row return:
  - `namespace = "default"`
  - `tags = []`
- No backfill is required; metadata is created/updated lazily on create/edit/bulk update.

## Validation & Normalization Rules

Applied in Console (shared by internal handlers + external routes):

- `namespace`
  - Default: `default`
  - Normalization: `trim()` then `toLowerCase()`
  - Max length: 64
  - Allowed pattern: `^[a-zA-Z0-9][a-zA-Z0-9._:/-]*$`
- `tags`
  - Default: `[]`
  - Normalization per tag: `trim()` then `toLowerCase()`
  - Max tag length: 32
  - Max count: 50
  - Allowed pattern: `^[a-zA-Z0-9][a-zA-Z0-9._-]*$`
  - Deduped

## Bulk Update Semantics (Important)

### Permissions

- `permissionsMode = replace`
  - Each target token’s permission set is replaced entirely with the submitted set.
- `permissionsMode = merge`
  - The submitted permissions are merged into existing permissions:
    - For each `serverId`, top-level fields (e.g., `enabled`) are overwritten if present in the patch.
    - Nested `tools` and `resources` maps are merged key-by-key; only submitted keys are overwritten.
    - Keys not present in the patch are preserved.

### Tags

- `tagsMode = replace`: overwrite tags with the submitted list
- `tagsMode = add`: add submitted tags (deduped)
- `tagsMode = remove`: remove submitted tags
- `tagsMode = clear`: remove all tags (no `tags` array required)

## UX / UI Requirements (Console)

### Console UI Implementation Details (Where/What to Change)

This section maps the UX requirements to the **current Console codebase** and calls out the exact files/areas to modify.

#### Access Token Management Page

File: `kimbap-console/app/(dashboard)/dashboard/members/page.tsx`

**1) Token data mapping**
- In the token list fetch (`api.tokens.getAccessTokens` → `tokenList`), extend the UI token shape to include:
  - `namespace` (string)
  - `tags` (string[])
- Ensure the transformation function includes sensible defaults:
  - `namespace = "default"` when missing
  - `tags = []` when missing
  - In the current file, this should be done inside the token mapping helper (e.g., `transformTokenList`), and reused anywhere the list is refreshed (initial load + after create/edit/bulk).

**2) Token row display**
- In the token row “First Row - Main Info” block (where token mask/name/role badges are rendered), add:
  - A namespace badge (e.g., `ns:default`)
  - Tag badges (show first 2 + overflow badge)

**3) Create Token Dialog**
- In the “Basic Information” area inside the Create Token dialog (after Notes), add inputs:
  - `Namespace` (default `"default"`)
  - `Tags` (comma-separated string)
- When submitting `operateAccessToken(handleType=1)`, include:
  - `namespace`
  - `tags` (parsed from comma-separated input)
  - In the current file, this is done in the `performCreateToken(...)` request payload.

**4) Edit Token Dialog**
- The members page uses `EditTokenDialog` for per-token edits; ensure:
  - The token object passed into `EditTokenDialog` includes `namespace/tags`
  - The `handleEditToken` → `operateAccessToken(handleType=2)` request includes `namespace/tags`
  - In the current file, the server call is performed inside `performEditToken(...)`.

**5) Selection + Bulk Update entry point**
- Add per-row checkboxes (Owner/Admin only) to select tokens for bulk actions.
- Add a “Select all manageable” checkbox row above the list.
- Add a “Bulk Update” button that opens the bulk dialog; disabled when no tokens are selected.
- Owner tokens (role=1) must not be bulk-editable (checkbox disabled or not shown).

**6) Bulk Update Dialog**
- Add a dialog that supports:
  - Permission copy source: choose a “source token” from the visible list
  - Permissions mode: `replace | merge`
  - Namespace input (blank = no change)
  - Tags mode: `add | remove | replace | clear`
  - Tags input (required unless mode is `clear`)
- On submit, call `operateAccessToken(handleType=4)` with:
  - `userids = selected token IDs`
  - `permissions` = permissions from the chosen source token (if selected)
  - `permissionsMode`
  - `namespace` (if provided)
  - `tagsMode` (+ `tags` if not `clear`)
  - In the current file, the handler should be a single function (e.g., `performBulkUpdateTokens`) that builds the payload conditionally.
- After completion:
  - Refresh the token list via `getAccessTokens`
  - Clear selection
  - Show success/failure toast and log failures for troubleshooting

#### Edit Token Dialog Component

File: `kimbap-console/components/edit-token-dialog.tsx`

- Extend the token prop shape to include:
  - `namespace?: string`
  - `tags?: string[]`
- Add inputs for namespace and tags (comma-separated) in the “Basic Information” section.
- On save, include `namespace` and parsed `tags` in the `updatedToken` passed to `onSave`.

### Token List (Access Token Management)

- Display per token:
  - Token mask + name + role (existing)
  - **Namespace badge** (e.g., `ns:default`)
  - **Tag badges** (show first 2 + `+N` overflow)
- Selection:
  - Owner/Admin can select tokens (checkbox per row)
  - “Select all manageable” checkbox + selected counter
  - Owner token (role=1) is excluded from bulk update selection (checkbox disabled or not shown)
- Bulk actions:
  - “Bulk Update” opens a dialog; disabled when nothing selected

### Create Token Dialog

Add metadata fields:
- `Namespace` input (default `default`)
- `Tags` input (comma-separated)

When creating a token:
- Permissions are set as usual via Scopes UI.
- Metadata is stored in Console DB for the created token.

### Edit Token Dialog

Add metadata fields:
- `Namespace` input
- `Tags` input (comma-separated)

On save:
- Update Core fields (name/notes/permissions) as before
- Update Console metadata (`namespace/tags`) for that token

### Bulk Update Dialog

Bulk update applies to selected tokens and supports:

1) **Permissions**
- Copy from an existing token (“source token”)
- Mode:
  - `replace`: overwrite permissions for each selected token
  - `merge`: merge submitted permissions into existing permissions

2) **Metadata**
- Namespace: optional (blank = no change)
- Tags:
  - Mode: `add | remove | replace | clear`
  - Tags input required unless `clear`

Post-action:
- Refresh list and clear selection
- Show failure count if any token update fails

## Internal API (Console `/api/v1` Protocols)

### Protocol 10007 (List Tokens)

Response `tokenList[]` adds:
- `namespace: string`
- `tags: string[]`

Behavior:
- Console fetches token list from Core, then joins metadata from Console DB.
- If no metadata row exists, defaults are returned (`default`, `[]`).

### Protocol 10008 (Operate Token)

#### Create (handleType=1)

Request adds optional:
- `namespace?: string`
- `tags?: string[]`

Behavior:
- Create token in Core
- Store plaintext token in Console `user` table (existing behavior)
- Upsert metadata row in `token_metadata`

Notes:
- This flow keeps the existing Console requirement to provide `masterPwd` for token creation (used to decrypt the owner token).

#### Edit (handleType=2)

Request adds optional:
- `namespace?: string`
- `tags?: string[]`

Behavior:
- Update Core (name/notes/permissions) as before
- If metadata provided, upsert metadata row (requires `proxyId` or a proxyId resolvable from the Core user record)

Notes:
- `permissions` updates in handleType=2 are **full replacement** (no partial patch). Use bulk update with `permissionsMode=merge` for merge semantics.
- `tags` updates in handleType=2 are **full replacement** (no add/remove/clear modes). Use bulk update with `tagsMode` for tag operations.

#### Delete (handleType=3)

Behavior:
- Delete in Core + disable user (existing behavior)
- Best-effort cleanup:
  - Delete metadata row from `token_metadata`
  - Delete local cache row from Console `user` table

#### Bulk Update (handleType=4)

Request:
```json
{
  "handleType": 4,
  "proxyId": 1,
  "userids": ["u1", "u2"],
  "permissionsMode": "replace",
  "permissions": [/* Tool[] from token list */],
  "namespace": "team-a",
  "tagsMode": "add",
  "tags": ["ci", "prod"]
}
```

Behavior:
- For each `userid`:
  - Skip/fail owner tokens (role=1)
  - Update Core permissions (replace or merge)
  - Update Console metadata (namespace/tags) via upsert
- Returns counts + per-token failures.

Notes:
- Bulk update does **not** change `role`, `expiresAt`, or `rateLimit` (out of scope).
- Protocol 10008 retains a legacy response shape (`accessToken: string`). For bulk update it returns a placeholder (e.g., `"bulk"`) plus `updatedCount/failedCount/failures`.

## External REST API (`/api/external`)

### Responses include metadata

- `POST /api/external/tokens`
- `POST /api/external/tokens/get`

Both include:
- `namespace`
- `tags`

### Batch Create Tokens

- `POST /api/external/tokens/create`
- Each token input supports `namespace?: string`, `tags?: string[]`
- Response for each created token returns `namespace`, `tags` (normalized)

Notes:
- External batch create uses the Owner token from the `Authorization` header directly (no `masterPwd` required).

### Update Token

- `POST /api/external/tokens/update`
- Supports optional `namespace`, `tags` updates in Console DB

### Batch Update Tokens

- `POST /api/external/tokens/batch-update`
- Supports:
  - `tokenIds[]`
  - permissions (replace/merge)
  - namespace
  - tags (add/remove/replace/clear)

## Error Handling & Atomicity

- Bulk operations are **non-atomic**: failures for one token do not prevent updating other tokens.
- Per-token updates may also be **partially applied** (e.g., Core permissions updated but Console metadata write failed). A retry with the same payload must converge the token to the intended final state.
- Token create flows are cross-system (Core + Console DB). There is no distributed transaction; Console should:
  - Validate metadata inputs before use
  - Persist metadata as part of the create flow (best effort if needed)

## Resumability, Retries, and Real-World Edge Cases

This feature will be used heavily in real environments (many tokens, unstable networks, browser refreshes). The design must assume operations can be interrupted and must be safe to retry.

### Idempotency guarantees (what can be safely retried)

- **Bulk update** (`/api/v1` protocol 10008 `handleType=4`, and `/api/external/tokens/batch-update`) must be **safe to retry** with the same payload:
  - `permissionsMode=replace` is idempotent.
  - `permissionsMode=merge` is idempotent as long as the submitted patch contains explicit `enabled` booleans (reapplying overwrites the same keys with the same values).
  - `tagsMode=add/remove/replace/clear` are idempotent by definition (dedupe + deterministic transforms).
  - `namespace` set is idempotent.
- **Single-token metadata updates** (`handleType=2` with `namespace/tags`) are idempotent (full replacement semantics).

### UI behavior for “stopped halfway”

In `kimbap-console/app/(dashboard)/dashboard/members/page.tsx`:

- Bulk update should show clear outcomes:
  - `updatedCount`
  - `failedCount`
  - per-token failures list (at least in an expandable detail view)
- Provide a **Retry Failed** action:
  - Select only failed token IDs and re-run the same bulk update payload.
- If the browser is refreshed or the request times out:
  - The user must be able to safely re-run the same bulk update on the same selection (idempotency).
  - For large batches, the UI should send requests in **chunks** (e.g., 50–200 token IDs per request) and show a progress indicator; stopping/canceling should stop sending further chunks, but already-applied updates remain.

### API behavior for “client retry due to timeout”

- Bulk update endpoints should return:
  - `updatedCount`, `failedCount`, `failures[]` (already defined)
- Clients should be encouraged to:
  - retry only failures when a response is received
  - retry the whole request if the client lost the response (safe due to idempotency)

### Common edge cases to handle explicitly

- **Owner token included** in a bulk update selection:
  - Must be rejected per token (reported in failures), not updated.
- **Token deleted** while bulk update is running:
  - Must fail for that token only; other tokens continue.
- **Mixed proxy IDs** or stale token list:
  - Internal bulk update requires `proxyId`; the server must ensure the target tokens belong to that proxy (otherwise fail per token).
- **Core unavailable / transient errors**:
  - Partial failures are expected; results must reflect which tokens failed so the user can retry.
- **Large batch timeouts**:
  - UI chunking + retry is required; do not assume a single request can update thousands of tokens reliably.
- **Concurrent updates** by multiple admins:
  - Last write wins; results may differ if the underlying permissions changed during the run. Retrying with the same payload should converge to the desired final state.

## Implementation Touchpoints (Console Repo)

### Database / Prisma

- `kimbap-console/prisma/schema.prisma`
  - Add `TokenMetadata` model mapped to `token_metadata`.
- `kimbap-console/prisma/migrations/20260213190000_add_token_metadata/migration.sql`
  - Create `token_metadata` table + index.

### Shared metadata logic

- `kimbap-console/lib/token-metadata.ts`
  - Normalization/validation
  - DB helpers (`getTokenMetadataMap`, `upsertTokenMetadata`, `deleteTokenMetadata`)
  - Tag operation helper (`applyTagsOperation`)

### Internal Protocol Handlers

- `kimbap-console/app/api/v1/handlers/protocol-10007.ts`
  - Add `namespace/tags` to response.
- `kimbap-console/app/api/v1/handlers/protocol-10008.ts`
  - Accept `namespace/tags`
  - Upsert metadata on create/edit
  - Cleanup metadata on delete
  - Add `handleType=4` bulk update

### Internal API Client

- `kimbap-console/lib/api-client.ts`
  - Extend `operateAccessToken` params for metadata + bulk update.

### External API Routes

- `kimbap-console/app/api/external/tokens/route.ts`
  - Add metadata fields in list response.
- `kimbap-console/app/api/external/tokens/get/route.ts`
  - Add metadata fields in get response.
- `kimbap-console/app/api/external/tokens/create/route.ts`
  - Accept metadata in inputs, persist to Console DB, return metadata.
- `kimbap-console/app/api/external/tokens/update/route.ts`
  - Accept metadata updates.
- `kimbap-console/app/api/external/tokens/delete/route.ts`
  - Cleanup metadata + local cache row.
- `kimbap-console/app/api/external/tokens/batch-update/route.ts`
  - New endpoint for batch update.
- `kimbap-console/app/api/external/API.md`
  - Update docs for metadata + batch update.

### Console UI

- `kimbap-console/app/(dashboard)/dashboard/members/page.tsx`
  - Show namespace/tags
  - Create token metadata inputs
  - Bulk update dialog + selection UX
- `kimbap-console/components/edit-token-dialog.tsx`
  - Edit token metadata inputs

## Acceptance Criteria

- Creating a token from Console UI persists:
  - Core user record + permissions
  - Console `token_metadata` row (namespace/tags)
- Listing tokens shows `namespace/tags` and defaults for legacy tokens.
- Editing a token updates `namespace/tags` and reflects immediately in the list.
- Bulk update from Console UI can:
  - Copy permissions from a source token to N selected tokens
  - Update namespace/tags for N selected tokens
  - Report failures without aborting the entire batch
- External API:
  - Batch create accepts metadata and returns it
  - Batch update endpoint works and is documented

## Verification / Commands

From `kimbap-console/`:

```bash
npm run db:generate
npm run type-check
npm run lint
```
