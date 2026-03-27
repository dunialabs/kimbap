# Contributing Guide

Thank you for considering contributing to Kimbap.

## How to Contribute

### Reporting Issues

If you find a bug or have a feature request:

1. Check existing issues first.
2. Open a new issue with:
   - Clear title and description
   - Reproduction steps (if applicable)
   - Expected behavior vs actual behavior
   - Environment details (Go version, OS, deployment mode)
   - Relevant logs or screenshots

### Submitting Code

1. **Fork and clone**
   ```bash
   git clone https://github.com/your-username/kimbap.git
   cd kimbap
   ```

2. **Create a branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/your-bug-fix
   ```

3. **Set up development environment**
   ```bash
   make deps
   make dev
   ```

4. **Make changes**
   - Follow existing style and Go conventions
   - Keep changes scoped and readable
   - Do not hardcode secrets
   - Use structured logging via zerolog wrappers
   - Update relevant documentation (`README.md`, `CLAUDE.md`, `docs/`)
   - Add tests where practical

5. **Verify locally**
   ```bash
   make test
   go build ./...
   ```

6. **Commit changes**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

   Commit message convention:
   - `feat:` new feature
   - `fix:` bug fix
   - `docs:` documentation update
   - `refactor:` code refactoring
   - `test:` tests
   - `chore:` tooling or maintenance

7. **Push and open PR**
   ```bash
   git push origin feature/your-feature-name
   ```

   In the PR:
   - Explain what changed and why
   - Link related issues
   - Include validation notes (tests/build)

## Code Standards

- Keep functions and modules focused and cohesive
- Prefer explicit error handling and contextual logs
- Preserve auth, policy, and request-routing semantics
- Ensure cleanup paths are safe and complete

## Canonical Patterns

### Adding a new feature to Kimbap

**API endpoints:** Add new endpoints to the REST v1 API (`/v1`):
- Routes go in `internal/api/routes.go`
- Handlers go in `internal/api/handlers.go`
- Follow existing RESTful resource patterns (see tokens, policies, approvals)
- Use scope-based authorization: `r.With(RequireScope("resource:action"))`

### Adding a new service integration

Create a YAML file in `skills/official/`. Three adapter types are supported.

**HTTP (REST API — default)**

```yaml
name: service-name
version: 1.0.0
description: Short description
base_url: https://api.example.com
auth:
  type: bearer                    # none | header | bearer | basic | query | body
  credential_ref: service.token   # vault key reference
  query_param: api_key            # only used when type: query
actions:
  action-name:
    method: GET
    path: /resource/{id}
    description: What it does
    args:
      - name: id
        type: string
        required: true
    idempotent: true
    risk:
      level: low                  # low | medium | high | critical
```

**Command (CLI subprocess)**

```yaml
name: tool-name
version: 1.0.0
description: Wraps a CLI tool
adapter: command
auth:
  type: none                      # none | bearer (injects as KIMBAP_CREDENTIAL_<REF>)
command_spec:
  executable: cli-anything-tool   # must be on PATH or absolute
  json_flag: "--json"             # flag that makes the CLI emit JSON to stdout
  timeout: "60s"                  # optional per-service timeout
  env_inject:                     # optional extra env vars for the subprocess
    TOOL_ENV: production
actions:
  action-name:
    command: "tool subcommand"    # command + args passed before json_flag
    description: What it does
    args:
      - name: query
        type: string
        required: true
    risk:
      level: low
```

**AppleScript (macOS native apps)**

```yaml
name: app-name
version: 1.0.0
description: macOS Shortcuts automation
adapter: applescript
auth:
  type: none
target_app: AppName              # macOS app display name
actions:
  list-items:
    command: app-name-list-items  # must match a registered AppleScript command key
    description: List items
    risk:
      level: low
```

**Action naming convention:** Use kebab-case within the service file (e.g., `list-repos`, `create-issue`). The canonical name becomes `service.action-name` (e.g., `github.create-issue`).

### Adding a New Provider

Providers define OAuth endpoints and authentication configuration for external services. They are stored as YAML files in `internal/connectors/providers/official/`.

#### Steps

1. Copy `internal/connectors/providers/official/TEMPLATE.yaml` to `{provider-id}.yaml`
2. Fill in all required fields (see template comments)
3. Choose the appropriate `auth_lanes`:
   - `public-client` — embeds `client_id` in the binary (device/PKCE flows only; no secret)
   - `managed-confidential` — platform manages a registered app; `client_secret` stored in vault at `connector:{id}:client_secret`
   - `byo` — users provide their own credentials via `KIMBAP_{PROVIDER}_CLIENT_ID` / `KIMBAP_{PROVIDER}_CLIENT_SECRET` env vars
4. Set `token_exchange.auth_method` (usually `body`; use `basic` for providers like Notion or Stripe that require HTTP Basic auth)
5. Run `go test ./internal/connectors/providers/... -v` to verify parsing and parity

#### Auth Lanes Summary

| Lane | client_id source | client_secret source | When to use |
|------|-----------------|---------------------|-------------|
| `public-client` | `embedded_client_id` in YAML | None | Device/PKCE flows; public clients |
| `managed-confidential` | `managed_client_id` in YAML | Vault: `connector:{id}:client_secret` | Platform-operated app |
| `byo` | `KIMBAP_{ID}_CLIENT_ID` env | `KIMBAP_{ID}_CLIENT_SECRET` env | Enterprise/self-hosted |

Never put secrets in YAML files or source code.

### Environment variables

- Kimbap vars use the `KIMBAP_*` prefix where possible
- See `~/.kimbap/config.yaml` for all configuration options

## Testing

Run:

```bash
make test
go test ./...
```

## Security Issues

If you discover a security vulnerability, do not open a public issue. Contact the project maintainers directly with impact and reproduction details.

## License

By contributing code, you agree that your contributions are licensed under the [MIT License](https://opensource.org/licenses/MIT).
