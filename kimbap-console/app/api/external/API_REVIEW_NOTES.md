# Review Notes — KIMBAP Console External REST API Documentation (`/api/external`)

This document contains **review comments only** (no in-place edits) for `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/API.md`.

## Scope & Method

- Reviewed `API.md` content for correctness, consistency, and usability.
- Cross-checked key behaviors against the current implementation in `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/**/route.ts` and shared helpers in `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/lib/*`.

## High-Priority Issues (Correctness)

### 1) Rate limit unit is inconsistent with the product/UI

- `API.md` says:
  - Default rate limit: **1000 requests per hour**
  - Owner tokens: **10000 requests per hour**
  - Responses include `X-RateLimit-*` headers
- In the Console UI, rate limit is displayed as **`/min`** (per minute). Example: `/Users/brian/Desktop/kimbap/kimbap-console/app/(dashboard)/dashboard/members/page.tsx` shows `Rate Limit: {token.rateLimit}/min`.
- The code path that creates tokens defaults `ratelimit` to a small number (`10`) when `rateLimit` is not provided: `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/tokens/create/route.ts`.
- No code in this repo sets `X-RateLimit-*` headers for the external API responses (search for `X-RateLimit` only hits the doc).

**Recommendation (doc-level):**
- Define the unit precisely (per minute vs per hour) and ensure it matches the UI and backend enforcement.
- If rate-limit headers are not actually emitted by `/api/external`, either remove that claim or scope it to the actual API that emits them.

### 2) `POST /api/external/ip-whitelist/delete` response example does not match implementation

- `API.md` example response includes:
  - `deleted: 2`
  - `message: "IPs removed from whitelist"`
- Implementation returns:
  - `message: "Deleted X IP(s), Y failed"`
  - `results: [{ ip, success, error? }, ...]`
  - No `deleted` field
  - Also validates each element and may return `E1003` for invalid items

Source: `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/ip-whitelist/delete/route.ts`.

**Recommendation (doc-level):**
- Update the response example and include `results`.
- Add `E1003` to the endpoint’s error responses (invalid IP entry format).

### 3) `POST /api/external/kimbap-core/connect` “required fields” and response semantics are off

- `API.md` lists `host / port` as required in the error table.
- Implementation treats `port` as optional and validates it only if provided.
- `API.md` response shows `isValid: 1/2/3`, but implementation only returns `isValid: 1` in success responses; failures are thrown as errors (e.g., `E4014`, `E4015`) rather than returning `isValid: 2/3`.

Source: `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/kimbap-core/connect/route.ts`.

**Recommendation (doc-level):**
- Mark `port` as optional and document the defaulting rules.
- Either (a) remove `isValid: 2/3` from the response contract, or (b) document that failures are represented as error responses, not as `isValid` variants.

### 4) Error code catalog / “Error response by endpoint” section is outdated vs current external API

`API.md` includes many codes and endpoint-specific error mappings that do not correspond to the current `/api/external` code:

- Validation codes listed (e.g., `E1002`, `E1006`–`E1010`) are not defined in `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/lib/error-codes.ts`.
- The “Error response by endpoint” section mentions behaviors like `masterPwd` required for `/tools/enable` and `/tools/disable`, but current implementation does **not** accept `masterPwd` for these endpoints.
- The catalog includes DNS/tunnel/backup/license codes that are not part of the current external API surface (at least not implemented under `app/api/external`).

**Recommendation (doc-level):**
- Rebuild the error-code catalog from the actual external API error-code module and the route handlers.
- Keep the catalog narrowly scoped to what `/api/external` actually returns today.

### 5) `E4014` is documented as “KIMBAP Core not available” but used differently in multiple routes

In docs, `E4014` is “KIMBAP Core not available”. In code, `E4014` is thrown for cases like:
- “Cannot get access token for user: {tokenId}”

Sources:
- `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/user-servers/configure/route.ts`
- `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/user-servers/unconfigure/route.ts`
- `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/sessions/route.ts`
- `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/user-capabilities/set/route.ts`

**Recommendation (doc-level):**
- Align error code meaning with actual usage:
  - Either define a new error code for “token secret not found locally”, or
  - Adjust docs to describe `E4014` as a broader “external dependency / token secret unavailable” error (less ideal).

## Medium-Priority Issues (Clarity & Consistency)

### 1) Public `/proxy` endpoint returns sensitive data

- `/api/external/proxy` is explicitly “No authentication required” and returns `proxyKey`.
- This is likely sensitive; doc should include an explicit warning about exposure and recommended network/IP protections.

Source: `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/proxy/route.ts`.

### 2) Boolean fields use mixed representations (`true/false` vs `0/1`)

- Requests commonly use booleans (e.g., `allowUserInput: false`).
- Some responses return `allowUserInput: 0/1` (e.g., tool details).

Source: `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/tools/get/route.ts`.

**Recommendation (doc-level):**
- Add a short “Conventions” section clarifying this, or normalize to booleans everywhere (if the API intends to be purely REST/JSON).

### 3) Placeholder formats can be more precise

- `tokenId` and `proxyKey` are shown as `uuid-string`, but in practice they are **32-character lowercase hex** strings (SHA-256 truncated or UUID-without-hyphens).

Sources:
- `/Users/brian/Desktop/kimbap/kimbap-console/lib/crypto.ts` (`calculateUserId`)
- `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/auth/init/route.ts` (proxyKey generation)

### 4) “Server status” in `/proxy` is hard-coded

- `status` in `/proxy` is currently hard-coded to `1` with a `TODO`.

Source: `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/proxy/route.ts`.

**Recommendation (doc-level):**
- Note that `status` may be provisional or always `1` until implemented, to prevent integrator confusion.

## Suggested Doc Improvements (Structure)

- Add a small **“Conventions”** section near the top:
  - Content type, auth model, timestamp unit (seconds vs ms), empty-body POSTs (`{}`), boolean compatibility, and base-url handling.
- Separate “API contract” from “internal/legacy” sections:
  - Keep `/api/external` as the primary contract.
  - Move any legacy/unused error-code lists (DNS/backup/tunnel) to a separate appendix or remove if not implemented.
- For endpoints that proxy to Core and may return Core-level errors:
  - Explicitly document whether the external API passes through Core errors, maps them, or returns a generic internal error.

## Quick Consistency Checklist (What’s Good)

- Endpoint summary table exists and (for the current doc) matches the endpoint sections.
- Most routes consistently use the `{ success, data }` / `{ success, error }` envelope.
- Auth model (Owner-only token) matches `/Users/brian/Desktop/kimbap/kimbap-console/app/api/external/lib/auth.ts`.

