# Security

## Vault Encryption Model

Kimbap Core is designed for environments where secret material and control must stay inside your own infrastructure. The vault uses a password-based key derivation + authenticated encryption scheme.

### Key derivation (PBKDF2)

- Encryption keys are derived from a secret value (for example, a Kimbap access token) using PBKDF2 (HMAC-SHA-256) with a per-record random salt.
- The salt is at least 128 bits of randomness and is stored alongside the ciphertext.
- A high iteration count (on the order of 100k+ iterations) is used to make brute-force attempts significantly more expensive.
- The result is a 256-bit key suitable for AES-256-GCM.

### Authenticated encryption (AES-GCM)

- Secret values are encrypted with AES-256-GCM using a fresh IV/nonce for each encryption operation.
- AES-GCM produces both ciphertext and a 16-byte authentication tag.
- On decryption, the authentication tag is verified; if any part of the stored data has been modified, decryption fails and the value is rejected.

### What is stored at rest

For each encrypted secret, the database only stores:

- `salt` (for PBKDF2)
- `iv` / `nonce` (for AES-GCM)
- `ciphertext`
- `authTag`

The input secret and the derived AES keys never leave process memory and are not written to disk. In production, treat any secrets that can decrypt stored configuration blobs as high-value keys: provision them securely, avoid source control, and rotate them according to your organization's security policies.

---

## OAuth & Token Brokerage

Kimbap Core handles OAuth credentials for downstream services:

- **Downstream connector OAuth credentials (third-party providers).** Used to call external APIs on behalf of the agent. Kimbap Core stores these encrypted at rest (including refresh tokens where applicable), refreshes access tokens server-side, and injects only the access token into the execution context for the duration of the call.

**Security properties**:

- Refresh tokens and client secrets for downstream providers are never forwarded to callers.
- Long-lived credentials remain inside Kimbap Core; agents receive only a Kimbap bearer token.

---

## Policy Engine

Kimbap Core's policy system is how operators control what agents can do. Policy evaluation runs at stage 3 of the execution pipeline, before any credential is touched.

### How It Works

Policy rules are YAML documents stored in `internal/policy/`. Each rule matches on some combination of:

- caller identity (agent, token, role)
- action identifier (`service.action`)
- parameter values (e.g., block deletes on production resources)
- time of day or rate limits

Every action call is evaluated against the applicable rules. The outcome is always one of:

- `allow` — proceed to credential injection and execution
- `deny` — return an error immediately, write an audit record
- `require_approval` — suspend execution, create an approval record, notify the operator

### Human Approval Gates

When policy marks an action `require_approval`, the runtime:

1. Creates an approval record with the full request context
2. Suspends execution
3. Notifies the operator via configured webhook channels (email, Slack, Telegram, generic webhook)
4. Waits for an explicit approve or deny decision
5. On approval: resumes the pipeline from the credential stage
6. On denial: returns an error to the caller
7. Records the full decision path in audit

Approval records include: caller identity, action, parameters (sanitized), policy rule that triggered the gate, operator who decided, timestamp, and outcome.

This lets agents run autonomously for routine tasks while keeping humans in control of higher-risk operations.

---
