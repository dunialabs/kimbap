# Kimbap Test Scenario Catalog

**Version:** 0.1  
**Language:** English  
**Purpose:** Exhaustive QA / integration / security scenario catalog derived from the Kimbap PRD.

## Scope

This catalog is derived from the current Kimbap product direction:

- Action Runtime is the canonical core.
- CLI, proxy, run, and REST are adapters into the same runtime.
- Service tokens and short-lived sessions are the identity model.
- Vault, policy, audit, approval, and skill execution are first-class runtime components.
- Tier 1 / Tier 2 / Tier 3 integration rules drive test strategy.
- Multi-tenant isolation, key hierarchy, and secret non-exposure are core security guarantees.

## How to use this document

- Treat each scenario as an integration or end-to-end test idea unless noted otherwise.
- Convert the highest-value scenarios into automated tests first.
- Use the concrete Brave Search example as the canonical "public REST API -> Kimbap action/CLI" demo.
- Re-run the full matrix after any change to vault, proxy, policy, approval, token handling, or skill schema.

## Concrete public API example to include in demos

Use **Brave Search API** as the concrete public API -> Kimbap action example.

Why this is a good fit:
- It is a real public developer API.
- It uses a simple header-based auth model.
- It maps cleanly to a Tier 1 Kimbap skill.
- It is useful for real agent workflows such as research, market scanning, documentation lookup, and trend monitoring.

### Example skill sketch

```yaml
name: brave-search
version: 1.0.0
base_url: https://api.search.brave.com/res/v1

auth:
  type: header
  header_name: X-Subscription-Token
  credential_ref: BRAVE_SEARCH_API_KEY

actions:
  web_search:
    method: GET
    path: /web/search
    description: Search the public web using Brave Search
    args:
      - name: q
        type: string
        required: true
      - name: count
        type: integer
        required: false
      - name: country
        type: string
        required: false
      - name: search_lang
        type: string
        required: false
    request:
      query:
        q: "{{ args.q }}"
        count: "{{ args.count }}"
        country: "{{ args.country }}"
        search_lang: "{{ args.search_lang }}"
    response:
      extract: ".web.results[] | {title, url, description}"
      type: array
    risk:
      level: low
      mutating: false
```

### Concrete scenario that product and demo teams can reuse

**Scenario:** Daily market-research agent for developer tools

A company runs a morning research agent called `agent-research-daily`.  
The agent must gather five fresh web results for queries like:

- `"AI coding agent launches this week"`
- `"developer tools funding 2026"`
- `"open-source RAG framework release notes"`

The company does **not** want the agent process to see or store the Brave API key.

The operator therefore:

1. installs the `brave-search` Kimbap skill,
2. stores `BRAVE_SEARCH_API_KEY` in the Kimbap vault,
3. creates a service token for `agent-research-daily`,
4. allows `brave_search.web_search` in policy for that agent,
5. launches the agent with `kimbap run -- python agent.py`, or calls the action directly with `kimbap call brave_search.web_search`.

Expected behavior:
- Kimbap injects `X-Subscription-Token` server-side.
- The agent gets only normalized search results.
- Audit shows the agent, action, request ID, latency, and status.
- The Brave API key never appears in agent env, logs, or traces.

---

## 0. Concrete example: Public API -> Kimbap action/CLI (Brave Search)

| ID | Concrete scenario | Expected result |
|---|---|---|
| BRV-001 | Operator stores a valid Brave Search API key in the vault as `BRAVE_SEARCH_API_KEY`, installs the `brave-search` skill, and runs `kimbap call brave_search.web_search --q "AI agent frameworks" --count 5` from embedded mode. | Kimbap injects the API key into `X-Subscription-Token`, calls Brave Web Search successfully, returns five normalized results, and writes a successful audit event. |
| BRV-002 | A research agent launched with `kimbap run -- python agent.py` performs a normal HTTPS request to the Brave Search endpoint instead of calling Kimbap directly. | Proxy mode intercepts the call, associates it with the agent identity, injects the header server-side, and the agent never sees the raw Brave API key. |
| BRV-003 | The agent attempts the same search before the operator has added `BRAVE_SEARCH_API_KEY` to the vault. | Kimbap fails before any outbound request with a missing-credential error that identifies the missing vault reference but does not reveal secret material. |
| BRV-004 | The operator accidentally stores an invalid Brave API key and runs `kimbap call brave_search.web_search --q "open source RAG"`. | Kimbap receives a 401/403-style auth failure, maps it to a credential error, preserves correlation IDs, and does not retry indefinitely. |
| BRV-005 | The agent submits an empty query string to `brave_search.web_search`. | Input validation fails locally, no outbound network request is made, and the audit event is recorded as `validation_failed`. |
| BRV-006 | The agent requests `--country JP --search-lang ja --count 3` for a Japanese query through the Brave Search skill. | Kimbap forwards the expected parameters, receives localized results, and the normalized response preserves the requested locale metadata. |
| BRV-007 | The agent repeatedly calls Brave Search fast enough to trigger upstream rate limiting. | Kimbap classifies the upstream 429 correctly, applies configured backoff behavior, records retries and final status in audit, and never leaks the Brave API key into logs. |
| BRV-008 | Tenant Acme and tenant Globex both install the same `brave-search` skill but store different Brave API keys. | The same action ID resolves to different vault material per tenant, and cross-tenant calls never use the wrong subscription token. |
| BRV-009 | A policy allows `brave_search.web_search` for `agent-research` but denies all external search for `agent-finance`. | The research agent succeeds and the finance agent is denied before execution, proving action-level policy control across the same skill. |
| BRV-010 | An operator rotates the Brave API key in the vault while long-running research agents continue to search through connected mode. | New executions use the rotated key without requiring agent changes, and in-flight requests either complete safely or fail cleanly without mixed-key behavior. |
| BRV-011 | The operator exports audit data after a day of Brave Search usage by multiple agents and tenants. | Audit records can be filtered by tenant, agent, service `brave_search`, action `web_search`, request ID, and status without exposing the secret header value. |
| BRV-012 | The agent uses `kimbap actions describe brave_search.web_search` before first use. | Kimbap shows concrete parameter schema, required credential reference, risk classification, and output shape derived from the skill. |

## 1. Installation, bootstrap, and first-run flows

| ID | Concrete scenario | Expected result |
|---|---|---|
| BOOT-001 | A first-time developer installs the Kimbap binary on macOS, unlocks local mode, stores one test secret through stdin, and executes one low-risk action within five minutes. | The full install-to-first-action flow succeeds within the target onboarding time. |
| BOOT-002 | A first-time developer installs Kimbap on Linux in an offline environment with a pre-bundled skill file. | The binary starts, local vault works, and the skill installs from a local path without internet dependency. |
| BOOT-003 | A developer runs `kimbap --help` and `kimbap actions list` before any configuration exists. | Commands complete with actionable onboarding guidance instead of stack traces or empty ambiguous output. |
| BOOT-004 | A developer attempts to use `kimbap vault set STRIPE_KEY sk_live_xxx` as an inline argument. | Kimbap rejects the inline secret input and instructs the user to use stdin or `--file`, protecting shell history. |
| BOOT-005 | A developer starts `kimbap serve` on a port already in use. | Kimbap exits with a clear port-conflict error and does not partially initialize runtime components. |
| BOOT-006 | A developer starts `kimbap proxy` without having installed the local CA certificate and points an HTTPS client at it. | The failure mode is explicit and documented, and the client error does not result in secret exposure or undefined proxy state. |
| BOOT-007 | A developer runs `kimbap actions list` immediately after a clean install with no skills installed. | The output clearly shows that no actions are available yet and points to `kimbap skill install` or built-in official skills. |
| BOOT-008 | A developer upgrades Kimbap from a previous version and reuses the existing local vault metadata. | The runtime migrates local metadata safely or refuses to start with a precise migration error and rollback guidance. |

## 2. Identity, service tokens, and session tokens

| ID | Concrete scenario | Expected result |
|---|---|---|
| AUTH-001 | An operator creates a service token for `agent-billing` with a 24-hour TTL in connected mode. | The token is displayed once, stored hashed, associated with the correct tenant and agent, and visible in token metadata without revealing the raw value. |
| AUTH-002 | An operator lists tokens after issuing tokens for three agents in one tenant. | The list view shows non-secret metadata including agent name, tenant, scopes, created-at, expiry, and last-used. |
| AUTH-003 | An operator inspects a token that has never been used. | The token metadata shows `last_used = null` or equivalent without causing an access path to the raw token. |
| AUTH-004 | An operator revokes a token while the corresponding agent is idle. | Subsequent calls using that token fail immediately with a revocation error and are recorded in audit. |
| AUTH-005 | An operator rotates a service token atomically while the old token is still active. | The new token is issued, the old token is revoked, audit records link the rotation, and there is no double-valid window longer than the designed overlap. |
| AUTH-006 | A headless agent authenticates to `kimbap serve` using a valid service token and receives a short-lived session token. | The server issues a session token with the expected TTL and binds it to the original service identity and tenant context. |
| AUTH-007 | A headless agent attempts to reuse an expired session token without re-authenticating. | The request is rejected as expired and the client is instructed to re-authenticate or re-exchange the service token. |
| AUTH-008 | An agent uses a service token that belongs to tenant Acme against a base URL configured for tenant Globex. | Kimbap resolves the agent to Acme only and never crosses the tenant boundary based on URL or request parameters. |
| AUTH-009 | A malformed bearer token is sent in `Proxy-Authorization` to proxy mode. | The proxy denies the request early, emits a safe auth failure audit event, and never attempts vault lookup. |
| AUTH-010 | An agent attempts to call an action after its service token has expired but before the local process refreshes cached auth context. | Kimbap detects the expiry, refuses execution, invalidates the stale auth context, and forces re-authentication. |
| AUTH-011 | An operator tries to create a token with an invalid TTL syntax such as `--ttl potato`. | CLI validation rejects the command before any token is created. |
| AUTH-012 | A user logs in interactively with `kimbap auth login --server URL` and then runs `kimbap auth status`. | The status output shows current server, active identity, tenant, and token/session state without revealing secrets. |

## 3. Vault, encryption, and key hierarchy

| ID | Concrete scenario | Expected result |
|---|---|---|
| VLT-001 | A secret is stored in tenant Acme and then retrieved in masked form with `kimbap vault get BRAVE_SEARCH_API_KEY`. | The default output is masked, includes metadata, and never prints the full secret. |
| VLT-002 | The operator uses `kimbap vault get BRAVE_SEARCH_API_KEY --reveal` in a controlled local debugging session. | The secret is revealed only in the local terminal context, the reveal action is audited, and redaction rules still apply to logs. |
| VLT-003 | Two tenants store the same vault key name `GITHUB_TOKEN` with different values. | Each tenant resolves its own value and the same key name does not collide across namespaces. |
| VLT-004 | A tenant KEK is rotated after hundreds of secrets already exist in that tenant vault. | Secrets remain readable after rewrapping/re-encryption and the token model remains unchanged. |
| VLT-005 | A service token is revoked after secrets were stored for that tenant. | Revocation does not force vault re-encryption and tenant secrets remain accessible to other valid identities in the same tenant. |
| VLT-006 | The operator tries to import a secret from a file path that does not exist. | Kimbap fails clearly without creating an empty or corrupted vault entry. |
| VLT-007 | The operator attempts to overwrite an existing secret without `--force` or equivalent confirmation. | Kimbap requires explicit overwrite semantics or confirmation before replacing the stored value. |
| VLT-008 | A process crash occurs between writing encrypted payload data and updating secret metadata. | The vault remains in a consistent state, either fully old or fully new, with no half-written entry. |
| VLT-009 | An operator lists vault entries in a tenant containing both OAuth tokens and API keys. | The list output only shows metadata such as key names, types, updated-at timestamps, and optional tags. |
| VLT-010 | A backup/restore cycle is performed for the local or connected vault storage. | Restored secrets remain decryptable only with the correct KEK material and preserve tenant boundaries. |
| VLT-011 | A wrong master password or wrong local unlock key is used in embedded mode. | Unlock fails deterministically and does not corrupt local vault state. |
| VLT-012 | A secret contains newline characters, unicode text, or a PEM block. | The value round-trips losslessly through store and retrieve operations without truncation or encoding corruption. |

## 4. Action registry, validation, and canonical runtime behavior

| ID | Concrete scenario | Expected result |
|---|---|---|
| ACT-001 | The operator installs a skill that defines `github.list_pull_requests` and runs `kimbap actions list --service github`. | The action is listed once with stable canonical naming and discoverable metadata. |
| ACT-002 | A developer runs `kimbap actions describe github.list_pull_requests` in embedded mode and connected mode. | Both modes return the same schema, risk classification, credential references, and examples. |
| ACT-003 | An agent calls a valid action with a missing required parameter. | Kimbap rejects locally with a structured validation error and no outbound call occurs. |
| ACT-004 | An agent passes an unknown parameter to a strict action schema. | The runtime rejects the extra parameter or flags it clearly depending on configured schema strictness. |
| ACT-005 | A mutating action is invoked without a required idempotency key. | The runtime rejects execution before any side effects occur. |
| ACT-006 | The same action is invoked through `kimbap call`, `kimbap serve`, and `kimbap run` with equivalent input. | All three paths produce semantically equivalent results, policy decisions, and audit metadata. |
| ACT-007 | An action returns a partial response that fails the skill’s output extraction rule. | Kimbap reports a response-extraction error rather than silently returning malformed output. |
| ACT-008 | Two skills attempt to register the same canonical action ID. | The registry detects the conflict and refuses installation or forces an explicit namespace resolution. |
| ACT-009 | A developer removes a skill and immediately runs `kimbap actions list`. | The removed action disappears from the registry and can no longer be executed. |
| ACT-010 | An action description references a credential that is not currently present in the vault. | `kimbap actions describe` shows the credential requirement without failing, while execution later fails cleanly until the secret exists. |

## 5. Tier 1 skills and REST adapter execution

| ID | Concrete scenario | Expected result |
|---|---|---|
| SK1-001 | A Tier 1 bearer-auth skill executes a successful GET request with no query parameters. | Kimbap injects the bearer token, normalizes the response, and records the external endpoint used. |
| SK1-002 | A Tier 1 custom-header skill uses `X-API-Key` instead of `Authorization`. | Kimbap injects the configured header name exactly and does not add unintended auth headers. |
| SK1-003 | A Tier 1 query-param skill places the credential in the query string for a legacy API. | The credential is added only to the outbound request and is redacted from logs and audit. |
| SK1-004 | A Tier 1 basic-auth skill uses separate username and password vault refs. | Kimbap composes the correct Authorization header server-side and never exposes the pair to the agent. |
| SK1-005 | A Tier 1 skill defines default query parameters and the agent overrides one of them. | The runtime applies deterministic merge rules and the final request matches the documented precedence. |
| SK1-006 | A Tier 1 skill uses path templating and the agent passes a value containing reserved URL characters. | Kimbap encodes the value correctly and hits the expected upstream path. |
| SK1-007 | A Tier 1 skill returns paginated results with a cursor and `max_pages = 3`. | The runtime follows the cursor exactly up to three pages and returns a merged normalized result. |
| SK1-008 | A Tier 1 skill defines an error mapping from 401 to `credential_expired`. | When the upstream returns 401, Kimbap classifies it as an auth/credential issue instead of a generic upstream failure. |
| SK1-009 | A Tier 1 skill defines retry-on-429 with exponential backoff. | The runtime retries the configured number of times, honors backoff intervals, and reports retry metadata. |
| SK1-010 | A skill file contains invalid YAML syntax during `kimbap skill install`. | Installation fails with a parser error that points to the line and field causing the problem. |
| SK1-011 | A skill file passes YAML parsing but violates the Kimbap skill schema. | Installation fails with a semantic validation error naming the missing or invalid field. |
| SK1-012 | A skill action body template references an argument that is not declared in `args`. | `kimbap skill validate` or install-time validation catches the dangling template reference. |
| SK1-013 | A skill contains two actions, one read-only and one mutating, and both are listed after install. | The registry preserves per-action metadata including mutating state and risk classification. |
| SK1-014 | The operator installs the same official skill version twice. | The second install is idempotent or yields a clear `already installed` message without duplicate registry entries. |
| SK1-015 | The operator upgrades a skill from version 1.0.0 to 1.1.0 and one action output schema changes. | Kimbap surfaces a compatibility warning or diff and retains a clear audit trail of the skill version change. |
| SK1-016 | A skill references a credential named `NOTION_KEY`, but policy later denies all Notion mutations for one agent. | Execution fails due to policy before credential injection, proving that policy gates happen first. |

## 6. Tier 2 connectors, OAuth, and refresh lifecycle

| ID | Concrete scenario | Expected result |
|---|---|---|
| OA2-001 | An operator runs `kimbap connector login gmail` and completes device flow successfully. | Kimbap stores the resulting access and refresh tokens securely and lists the connector as healthy. |
| OA2-002 | The operator starts device flow but never completes the browser step. | The pending login expires cleanly and no half-configured connector remains active. |
| OA2-003 | A connected Gmail action is executed with an access token that is about to expire. | Kimbap refreshes transparently before or during execution and the agent sees only a successful action result. |
| OA2-004 | The refresh token has been revoked upstream and the next Gmail action requires refresh. | Kimbap reports a connector re-authentication requirement without exposing token details to the agent. |
| OA2-005 | Two agents in the same tenant share one connector configuration for a service. | Both agents can execute within policy while audit still attributes each action to the correct agent. |
| OA2-006 | Two tenants configure the same OAuth connector name `gmail` with different accounts. | Refresh state, access tokens, and audit remain completely isolated by tenant. |
| OA2-007 | The operator runs `kimbap connector refresh gmail` manually on a healthy connector. | A fresh access token is obtained or refreshed state is confirmed and the operation is audited. |
| OA2-008 | A connector refresh attempt happens while another request is already refreshing the same token. | Only one refresh wins, the second caller reuses the new token, and no token-stampede occurs. |
| OA2-009 | Upstream OAuth returns an invalid_grant error during refresh. | Kimbap marks the connector unhealthy, stops repeated retries, and prompts for re-login. |
| OA2-010 | The operator runs `kimbap connector list` after several connectors have mixed health states. | The output accurately shows healthy, expiring soon, expired, and re-auth required connectors. |
| OA2-011 | A connector stores scopes narrower than required for an action. | The action fails with a scope/permission error that makes clear whether policy or upstream auth was the blocker. |
| OA2-012 | A long-running agent executes Google Calendar reads for 72 hours through connected mode. | No token expiry error is surfaced to the agent if refresh remains valid. |
| OA2-013 | OAuth refresh token storage is inspected through logs and audit after heavy connector use. | Refresh tokens never appear in logs, audit payloads, CLI output, or proxy-visible material. |
| OA2-014 | A connector login is repeated for the same service/account pair after rotating client credentials. | Old refresh state is replaced safely, new client credentials are honored, and audit shows connector rebind. |

## 7. Proxy mode and kimbap run

| ID | Concrete scenario | Expected result |
|---|---|---|
| PRX-001 | A Python agent using the standard `requests` library is launched with `kimbap run -- python agent.py` and makes an ordinary HTTPS GET to an API supported by a Kimbap skill. | The request succeeds without code changes and the agent never receives the service credential. |
| PRX-002 | A Node.js agent using environment-aware HTTP libraries is pointed at `HTTP_PROXY` and `HTTPS_PROXY`. | Kimbap intercepts supported traffic and attributes calls to the configured agent identity. |
| PRX-003 | A client makes plain HTTP (not HTTPS) requests through proxy mode. | The request succeeds without CA installation and Kimbap still applies action resolution, policy, and audit. |
| PRX-004 | A client makes HTTPS requests before the local CA certificate is trusted. | The TLS failure is explicit, no secrets are leaked, and the operator is guided to the correct trust-store setup. |
| PRX-005 | A client that performs certificate pinning is routed through Kimbap proxy. | The client rejects the MITM certificate, Kimbap records the unsupported-mode failure, and no false success is reported. |
| PRX-006 | A client uses WebSocket upgrade semantics through the proxy. | Kimbap clearly reports that WebSocket is unsupported in v1 instead of hanging indefinitely. |
| PRX-007 | A client uses HTTP/2 with multiplexed requests to the same origin. | The proxy correctly handles concurrent streams within supported limits and preserves per-request correlation IDs. |
| PRX-008 | A long-running agent uses `kimbap run` and then spawns child processes. | Only the intended subprocess tree inherits proxy settings or agent context according to design, and secrets are not injected more broadly than intended. |
| PRX-009 | A proxy request arrives with no recognizable mapping to a registered action. | Kimbap either denies the request or routes it to a safe generic handler based on policy, but never bypasses policy and audit. |
| PRX-010 | A proxy request body includes sensitive user content that should not be logged. | Kimbap logs high-level metadata while respecting body redaction rules. |
| PRX-011 | An agent rapidly performs many repeated proxy requests to the same host and path. | Hot-path caches improve performance while still honoring policy and tenant boundaries. |
| PRX-012 | A request contains both an existing placeholder API key and an upstream Authorization header set by the agent. | Kimbap applies deterministic precedence rules and ensures the agent-provided header cannot override the trusted injected credential unexpectedly. |
| PRX-013 | Proxy mode is started with a configured default agent token, but an explicit `Proxy-Authorization` header is also present. | Kimbap chooses the intended precedence and records which identity source was used. |
| PRX-014 | A proxy request times out upstream after credentials were injected into the outbound request. | The timeout is surfaced cleanly, the secret never appears in error logs, and retry semantics remain policy-aware. |
| PRX-015 | The operator restarts the proxy while agents are still running. | New requests fail or reconnect cleanly without corrupting agent identity or partial credential reuse. |
| PRX-016 | A proxy request attempts to reach a domain that is not present in any installed skill or allowed host policy. | Kimbap blocks the request according to policy and records the attempted host in audit. |

## 8. Connected mode server and REST API

| ID | Concrete scenario | Expected result |
|---|---|---|
| SRV-001 | An authenticated client executes an action through `POST /v1/actions/{service}/{action}:execute`. | The result is identical to the equivalent `kimbap call` invocation including metadata and policy outcome. |
| SRV-002 | An unauthenticated client calls the execute endpoint. | The server rejects the request before action resolution and does not leak whether the action exists. |
| SRV-003 | A client requests `GET /v1/actions` in a tenant with ten installed skills. | The API returns only the actions visible in that tenant and filters correctly by installed skill set. |
| SRV-004 | A client calls `POST /v1/actions/validate` with invalid parameters. | The API returns structured validation errors and no side effects occur. |
| SRV-005 | A client reads `GET /v1/vault` without sufficient policy privileges. | The server rejects the request and does not reveal vault key names. |
| SRV-006 | A token rotation request is issued through the REST API while another request is concurrently inspecting the same token. | The API returns consistent token state and no stale secret material is exposed. |
| SRV-007 | A client calls `POST /v1/policies:evaluate` with agent, tenant, and action context. | The server performs dry-run only and returns matched rules plus the final decision. |
| SRV-008 | A client fetches `GET /v1/approvals` while multiple requests are pending and some are already expired. | The API distinguishes pending, approved, rejected, expired, and completed items accurately. |
| SRV-009 | A client exports audit data for a narrow time window and one tenant. | The output is filtered correctly and never includes secret fields. |
| SRV-010 | A client invokes a non-existent typed endpoint such as `/v1/actions/foo` with an unsupported method. | The server returns a predictable 404/405 style error and does not fall back to generic admin action routing. |
| SRV-011 | A reverse proxy or API gateway strips a required auth header before forwarding to Kimbap serve. | Kimbap fails safely and does not interpret the request as anonymous local traffic. |
| SRV-012 | A connected deployment is restarted during a spike of valid action traffic. | Clients either reconnect or fail cleanly and the server comes back without registry or vault corruption. |

## 9. Policy engine and dry-run behavior

| ID | Concrete scenario | Expected result |
|---|---|---|
| POL-001 | A read-only policy allows `github.*` for `agent-research` only when `mutating=false`. | A `github.list_pull_requests` call succeeds and `github.create_issue` is denied for that same agent. |
| POL-002 | A policy explicitly denies `notion.delete_*` for all agents. | Delete actions are blocked before credential injection regardless of connector state. |
| POL-003 | A policy requires approval for `stripe.refund_charge` when amount exceeds a threshold in the action arguments. | Low-value refunds proceed and high-value refunds enter the approval queue. |
| POL-004 | A rate-limit policy allows only ten `gmail.send_message` actions per minute for one agent. | The eleventh call within the window is blocked or delayed according to design and recorded as a policy-rate-limit event. |
| POL-005 | A policy references a tenant-scoped tag such as `environment=prod`. | The same action is allowed in staging and denied in production for the same agent. |
| POL-006 | `kimbap policy eval --agent agent-billing --action stripe.refund_charge` is run for a policy that would require approval. | The command returns `require_approval` and explains why without creating an approval record. |
| POL-007 | A malformed policy file is applied with `kimbap policy set --file rules.yaml`. | The update is rejected atomically and the previous policy set remains active. |
| POL-008 | A policy file contains overlapping rules with conflicting outcomes. | Kimbap applies deterministic precedence rules and surfaces the effective resolution. |
| POL-009 | A policy attempts to match on an action that does not exist. | The policy can still be stored if syntactically valid, but dry-run and lint tooling warn about dead matches. |
| POL-010 | A policy change is applied while long-running agents are active in connected mode. | New requests see the new policy immediately or after the documented consistency window; existing in-flight executions are not misclassified retroactively. |
| POL-011 | A policy viewer in console shows read-only summaries while the CLI remains the write surface. | The console never silently edits live policy and always reflects the authoritative current revision. |
| POL-012 | An action is permitted by upstream service permissions but denied by Kimbap policy. | The final outcome is denial because Kimbap policy is enforced before execution. |
| POL-013 | An action is allowed by Kimbap policy but denied by the upstream service due to its own access controls. | The error is classified as upstream authorization failure, not a Kimbap policy failure. |
| POL-014 | A policy rule matches wildcard service names such as `github.*` across two installed GitHub skills versions. | Matching behavior remains stable across versions or the policy compiler requires version-aware targeting. |
| POL-015 | A policy attempts to match on a sensitive argument value that should later be redacted in audit. | Policy can use the value for decisioning while audit stores only the redacted representation. |
| POL-016 | A policy with time-of-day restrictions is evaluated across a DST boundary or timezone change. | The correct policy window is applied using the configured timezone semantics. |

## 10. Approval queue and HITL workflows

| ID | Concrete scenario | Expected result |
|---|---|---|
| APR-001 | `agent-billing` calls `stripe.refund_charge` for an amount that requires approval. | Kimbap creates one approval request, pauses execution, and returns a pending status with a request ID. |
| APR-002 | An operator approves the pending request from the CLI with `kimbap approve <request-id>`. | The original action resumes exactly once and a full audit chain links request, approval, and execution. |
| APR-003 | An operator denies the pending request from the CLI with a reason. | Execution never happens, the agent receives a denied result, and the denial reason is stored in audit. |
| APR-004 | A Slack adapter sends an approval message containing one-time signed Approve/Deny links. | Using one link completes the decision and reusing the same link later fails safely. |
| APR-005 | Two operators try to approve the same request at nearly the same time. | Only one decision wins and the other receives an `already resolved` response. |
| APR-006 | A pending approval reaches its expiry time with no human response. | The action transitions to `EXPIRED`, the agent is unblocked with an expiry outcome, and no late approval can resurrect it. |
| APR-007 | A webhook adapter is configured and the external approval receiver returns HTTP 500. | Kimbap retries or records delivery failure according to adapter policy, while the approval request itself remains durable. |
| APR-008 | A notification adapter is down but the console remains healthy. | Operators can still resolve approvals from console or CLI without dependency on the failed adapter. |
| APR-009 | A multi-approver rule requires two approvals before execution. | The action remains pending after the first approval and executes only after the second valid approval. |
| APR-010 | An approval request contains arguments with sensitive values, such as email content or API payload text. | Notifications and console views show redacted or minimized context according to policy. |
| APR-011 | An agent retries the same high-risk request with the same idempotency key while approval is still pending. | Kimbap reuses or links to the same approval request instead of creating duplicates. |
| APR-012 | The original requester token is revoked while its approval request is still pending. | Decisioning behavior follows policy: either allow completion for the original request or fail closed, but it is deterministic and auditable. |
| APR-013 | An approval is granted, but the upstream service call fails afterward. | The system records `approved but execution failed` distinctly from `denied` and `expired` outcomes. |
| APR-014 | An operator views the approval queue filtered by tenant and risk level in console. | Only the intended tenant’s pending approvals are visible and all counts match the underlying queue state. |

## 11. Audit, logging, and observability

| ID | Concrete scenario | Expected result |
|---|---|---|
| AUD-001 | A successful low-risk action is executed through `kimbap call`. | An audit event includes request ID, trace ID, agent, tenant, service, action, latency, and success status. |
| AUD-002 | A validation failure occurs before any network call. | Audit still records the attempt with `validation_failed` status and zero outbound target. |
| AUD-003 | A policy denial occurs before credential injection. | Audit shows the attempted action and denial reason but not any secret lookup result. |
| AUD-004 | A proxy request succeeds and the operator tails audit with `kimbap audit tail --service brave_search`. | The event appears in real time with correct service tagging. |
| AUD-005 | A user exports audit data as JSONL for a time range containing multiple tenants. | The export can be filtered and contains no raw credential values or refresh tokens. |
| AUD-006 | A user exports audit data as CSV for business review. | Flattened CSV columns preserve key metadata such as agent, service, action, status, and latency. |
| AUD-007 | A connector refresh occurs without any agent-visible error. | The refresh itself still produces an internal audit event or metric that operators can inspect. |
| AUD-008 | A log aggregation pipeline consumes structured Kimbap logs from connected mode. | Logs remain valid JSON and correlation IDs make it possible to stitch request lifecycle together. |
| AUD-009 | An upstream request body includes personally sensitive user content. | Configured redaction rules prevent raw content from being persisted in logs or notifications. |
| AUD-010 | An operator filters the console audit viewer by `agent=agent-research` and `service=brave_search`. | The view returns the same records as CLI/API export for equivalent filters. |

## 12. Multi-tenant isolation

| ID | Concrete scenario | Expected result |
|---|---|---|
| TEN-001 | Tenant Acme and tenant Globex both install the same official skill set and create agents with the same agent name `agent-research`. | Actions and audit remain isolated because tenant context is part of every identity and record. |
| TEN-002 | A token from tenant Acme tries to fetch token metadata for tenant Globex through the REST API. | The request is denied without revealing whether the target token exists. |
| TEN-003 | A proxy request from tenant A targets a host commonly used by tenant B and both have different credentials for the same service. | The outbound request uses only tenant A’s credential material. |
| TEN-004 | Tenant A rotates its KEK while tenant B is actively executing actions. | Tenant B is unaffected and tenant A remains readable after rekey. |
| TEN-005 | A connector refresh happens simultaneously in two tenants for the same service name. | Refresh state is isolated and no token or status crosses tenants. |
| TEN-006 | An audit export for tenant A is requested by a tenant B identity. | The export is denied and no cross-tenant metadata is disclosed. |
| TEN-007 | A shared connected deployment serves ten tenants and one tenant generates a flood of low-risk calls. | Rate limits or resource isolation prevent one tenant from starving others. |
| TEN-008 | A skill is upgraded in tenant A but intentionally not in tenant B. | Each tenant sees its installed version only, and action behavior remains version-correct. |
| TEN-009 | A tenant-local policy denies all `gmail.*` while another tenant allows them. | The two tenants observe their own policy outcomes for identical actions. |
| TEN-010 | A secret backup snapshot from tenant A is mistakenly restored into tenant B’s storage path. | Kimbap detects the tenant/key mismatch or decryption failure and refuses cross-tenant materialization. |
| TEN-011 | Console views are filtered by tenant for approvals, audit, and tokens. | Operators only see records for tenants they are authorized to manage. |
| TEN-012 | A service token is issued with scopes intended for one tenant and copied into another tenant’s environment by mistake. | Kimbap still resolves the request to the original tenant or rejects it; it never grants access to the receiving tenant. |

## 13. Existing CLI executor adapters

| ID | Concrete scenario | Expected result |
|---|---|---|
| CLI-001 | Kimbap wraps `gh` for a read-only GitHub action using a temporary isolated config directory. | The action succeeds, no persistent login state leaks into the user’s default home directory, and output is normalized. |
| CLI-002 | Kimbap wraps a CLI that insists on writing auth cache files into `$HOME`. | The wrapper redirects `$HOME` to an isolated temp directory and cleans it up after execution. |
| CLI-003 | A wrapped CLI exits with a non-zero code and prints helpful error text to stderr. | Kimbap captures the exit status, normalizes the error, redacts secrets, and records adapter metadata in audit. |
| CLI-004 | A wrapped CLI requires an interactive TTY prompt unexpectedly. | Kimbap detects unsupported interactive behavior and fails safely instead of hanging forever. |
| CLI-005 | A wrapped CLI receives a temporary credential through environment injection for one execution. | The credential is not visible to the upstream agent process and is removed after execution. |
| CLI-006 | A wrapped CLI returns machine-readable JSON on one version and formatted text on another version. | Kimbap either pins supported versions or fails validation if output is no longer parseable. |
| CLI-007 | A wrapped CLI action is subject to a Kimbap approval policy. | Approval occurs in Kimbap before the CLI process is spawned. |
| CLI-008 | A wrapped CLI attempts to call back into the network directly with its own credential persistence flow. | Kimbap sandboxing or wrapper rules prevent unsafe escape paths or mark the adapter unsupported. |
| CLI-009 | A wrapped CLI is upgraded by the operating system package manager without Kimbap being updated. | Compatibility checks or smoke tests detect whether the adapter still behaves as expected. |
| CLI-010 | A wrapped CLI action times out after spawning a subprocess tree. | Kimbap kills the full process tree, cleans up temp files, and records a timeout outcome. |

## 14. Security and adversarial scenarios

| ID | Concrete scenario | Expected result |
|---|---|---|
| SEC-001 | A prompt-injected agent tries to print all environment variables while running under `kimbap run`. | Raw service credentials are not present in the agent-visible environment. |
| SEC-002 | A prompt-injected agent asks the operator to run `kimbap vault get SECRET --reveal` and paste the result back. | The operating skill and docs discourage this path, and audit makes any reveal action visible to operators. |
| SEC-003 | An agent intentionally sends a malformed request hoping proxy mode will dump the full outbound headers in an exception trace. | Kimbap error handling redacts auth headers and secret query params from all trace output. |
| SEC-004 | A compromised agent tries to bypass Kimbap by talking directly to the external API host from the same machine. | In controlled deployments, egress restrictions or policy detect and reduce bypass opportunities; Kimbap-only mode is enforceable where configured. |
| SEC-005 | An attacker steals a service token from one machine and reuses it from another IP or environment. | Detection and policy hooks can flag unusual token usage, and rotation/revocation immediately cut off access. |
| SEC-006 | An attacker replays a previously captured session token after it has expired. | The replay fails and is recorded as an auth anomaly. |
| SEC-007 | An operator accidentally includes a secret in a policy YAML comment or action argument example. | Linting or secret scanning detects obvious secret material before policy/skill install when such tooling is enabled. |
| SEC-008 | An upstream API returns a body that echoes submitted credentials or sensitive inputs. | Redaction rules scrub the response before it reaches logs, audit, notifications, or optional console previews. |
| SEC-009 | A malicious community skill tries to exfiltrate secrets by mapping them into visible response fields or unexpected query params. | Skill review, provenance checks, install diffing, and policy restrictions surface the risky behavior before or during install. |
| SEC-010 | A malicious community skill changes the target base URL from the expected service host to an attacker-controlled host in an update. | Digest pinning and diff review reveal the host change before upgrade is applied. |
| SEC-011 | A tenant admin with limited privileges attempts to reveal another admin’s secrets through audit export. | Audit export remains metadata-only with secrets redacted, so secret retrieval is impossible through logs. |
| SEC-012 | An action argument includes shell metacharacters that would be dangerous if passed to a CLI adapter unsafely. | The adapter uses safe argument passing or rejects the input; no command injection occurs. |
| SEC-013 | A user tries to install a skill whose declared credential reference collides with a high-value existing vault key on purpose. | Install review or namespace rules surface the collision and prevent silent reuse in the wrong context. |
| SEC-014 | A local machine crash dump or panic report is collected after a proxy request. | Secret values are absent or redacted from panic strings and structured error payloads. |
| SEC-015 | A human approver clicks the same one-time signed approval URL multiple times or shares it publicly. | The first valid use resolves the request, subsequent uses fail, and link abuse is auditable. |
| SEC-016 | A proxy client supplies both an agent token and spoofed tenant headers hoping to escalate tenants. | Kimbap trusts only authoritative identity sources and ignores untrusted tenant claims. |
| SEC-017 | A user runs `ps` or shell history inspection while Kimbap commands are being used. | No raw secrets appear in process arguments or command history for recommended flows. |
| SEC-018 | An external vault provider is temporarily unavailable while Kimbap needs a secret fetch. | Kimbap fails safely or uses only documented short-lived cache material without broadening exposure. |

## 15. Failure injection, recovery, and chaos

| ID | Concrete scenario | Expected result |
|---|---|---|
| CHA-001 | The upstream API becomes unreachable due to DNS failure during a low-risk action. | Kimbap returns a classified upstream network error with retry metadata and preserves audit context. |
| CHA-002 | The upstream API returns intermittent 500 errors for a retryable read action. | Configured retries are attempted with backoff and the final result reflects both retry count and terminal status. |
| CHA-003 | The Kimbap process crashes immediately after creating an approval request but before delivering notifications. | On restart, the approval request still exists and can be delivered or resolved without duplication. |
| CHA-004 | The Kimbap process crashes after an approval is granted but before the downstream action is executed. | Recovery logic ensures the request is either executed exactly once or safely marked for operator reconciliation. |
| CHA-005 | The database or state store becomes read-only during token rotation. | Rotation fails atomically and neither old nor new token state becomes ambiguous. |
| CHA-006 | A disk-full condition occurs during audit export. | The export fails with a clear error and does not corrupt the live audit store. |
| CHA-007 | Clock skew exists between Kimbap and an upstream OAuth provider. | Expiry handling tolerates skew within a safe margin and avoids using already-expired tokens. |
| CHA-008 | Network partition isolates Kimbap from Slack while approvals still accumulate. | The approval queue remains durable and alternative resolution paths still work. |
| CHA-009 | A proxy worker panics while serving one request under heavy load. | The failed request is isolated, the worker recovers or restarts, and other requests continue without secret leakage. |
| CHA-010 | A hot skill reload or skill installation occurs while requests are in flight. | Existing requests finish with the version they started against and new requests see the new version after activation. |
| CHA-011 | A tenant KEK rotation is interrupted halfway through a batch process. | The system can resume or roll back without rendering secrets unreadable. |
| CHA-012 | The console goes down during an ongoing approval workflow. | CLI and API resolution paths remain available and approval state is unaffected. |

## 16. Performance, scale, and soak tests

| ID | Concrete scenario | Expected result |
|---|---|---|
| PERF-001 | A single tenant executes 100 low-risk read actions per second through `kimbap serve` for 30 minutes. | Latency, error rate, and audit completeness remain within target thresholds. |
| PERF-002 | Ten tenants each execute 20 actions per second with different skills and credentials. | Multi-tenant throughput remains fair and no tenant crosses another tenant’s cache or vault boundary. |
| PERF-003 | Proxy mode sustains a large number of concurrent simple GET requests to one Tier 1 service. | Hot-path caching reduces overhead and p99 latency remains acceptable. |
| PERF-004 | A 72-hour soak test continuously exercises one OAuth-backed connector with periodic refreshes. | No agent-visible token expiry failures occur if upstream auth remains valid. |
| PERF-005 | The approval queue receives thousands of requests while operators resolve them gradually. | Queue storage remains healthy, statuses remain accurate, and resolution latency stays predictable. |
| PERF-006 | Audit tailing is run continuously while high-volume actions are executed. | Streaming audit does not fall behind significantly or lose records. |
| PERF-007 | A large action registry with hundreds of installed actions is loaded at startup. | Startup time and memory growth remain acceptable and action lookup remains fast. |
| PERF-008 | A vault with tens of thousands of secrets is queried by many agents in connected mode. | Lookup latency remains stable and tenant-scoped indexing works correctly. |
| PERF-009 | Policy evaluation is benchmarked across large rule sets with wildcard matches and argument constraints. | `kimbap policy eval` and live execution remain within the p99 target. |
| PERF-010 | A skill with paginated aggregation fetches many pages under a high but allowed rate budget. | Kimbap respects per-service and per-agent limits while still completing within documented bounds. |

## 17. Upgrade, migration, and compatibility

| ID | Concrete scenario | Expected result |
|---|---|---|
| MIG-001 | A v0.5-era local config is read by the v0.6 runtime after the product rename and command taxonomy stabilization. | Migration occurs cleanly or the runtime emits a precise incompatible-config error with guidance. |
| MIG-002 | A skill authored against an older schema version is installed on a newer Kimbap build. | Compatibility rules accept, warn, or reject deterministically based on declared version support. |
| MIG-003 | A connected deployment is upgraded while agents are still using older client binaries. | Core action semantics remain backward-compatible or version negotiation is explicit. |
| MIG-004 | A token created before the introduction of session tokens is used against a newer server version. | The server handles legacy bootstrap correctly or forces an explicit migration path. |
| MIG-005 | An installed skill is rolled back from version 1.2.0 to 1.1.0 after a bad release. | The registry and action registry reflect the rollback cleanly and audit preserves version history. |
| MIG-006 | A tenant imports a private skill from GitHub and later moves it to a private registry. | The action IDs remain stable or the rename is surfaced as a managed migration. |
| MIG-007 | A future optional MCP adapter is enabled alongside existing Kimbap action use. | The MCP adapter does not change the semantics of existing CLI, REST, or proxy paths. |
| MIG-008 | A major Kimbap upgrade changes policy compiler internals but not the DSL. | Existing policy files still compile to equivalent decisions under regression tests. |
