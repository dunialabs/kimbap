# Contributing Guide

Thank you for considering contributing to Kimbap Core Go.

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
   git clone https://github.com/your-username/kimbap-core.git
   cd kimbap-core
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

### Adding a new feature to Kimbap Core

**API endpoints:** Add new endpoints to the REST v1 API (`/api/v1`):
- Routes go in `internal/api/routes.go`
- Handlers go in `internal/api/handlers.go`
- Follow existing RESTful resource patterns (see tokens, policies, approvals)
- Use scope-based authorization: `r.With(RequireScope("resource:action"))`

**Do NOT** add new handlers to `/admin` or `/user` — these are legacy and frozen.

### Adding a new service integration (Skill)

Create a YAML file in `skills/official/`:

```yaml
name: service-name
version: 1.0.0
description: Short description
base_url: https://api.example.com
auth:
  type: bearer                    # bearer | header | query_param | basic | oauth2
  credential_ref: service.token   # vault key reference
actions:
  action-name:
    method: GET
    path: /resource/{id}
    description: What it does
    args:
      - name: id
        type: string
        required: true
    risk:
      level: low                  # low | medium | high | critical
      mutating: false
```

**Action naming convention:** Use kebab-case within the skill file (e.g., `list-repos`, `create-issue`). The canonical name becomes `service.action-name` (e.g., `github.create-issue`).

### Environment variables

- Kimbap Core vars use the `KIMBAP_*` prefix where possible
- `KIMBAP_CORE_URL` is the canonical name for the Core connection URL
- `JWT_SECRET` must match between Core and Console

## Testing

Run:

```bash
make test
go test ./...
```

## Security Issues

If you discover a security vulnerability, do not open a public issue. Contact the project maintainers directly with impact and reproduction details.

## License

By contributing code, you agree that your contributions are licensed under the [Elastic License 2.0](https://www.elastic.co/licensing/elastic-license).
