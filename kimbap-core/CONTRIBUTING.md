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
