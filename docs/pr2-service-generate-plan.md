# PR2 Service Generate First-Class Onboarding Plan

## Roadmap Review

PR 2 should remain a standalone PR based on `main`, even after the catalog-discovery work.

Why this is still the right cut:

1. It is product-coherent on its own.
   - Everything in scope improves one workflow: turning an OpenAPI spec into an installable Kimbap service.

2. It does not depend on PR 1.
   - No catalog-discovery code is required.
   - No manifest schema change is required.
   - No runtime adapter change is required.

3. The cleanest implementation boundary is an OpenAPI generation options layer.
   - `--name`, `--tag`, and `--path-prefix` are generation concerns, not just CLI formatting concerns.
   - Doing this only in `cmd/kimbap/` would create post-generation hacks, especially for action-key determinism after filtering.

4. URL security policy should live in one place.
   - The CLI should stop pre-rejecting all `http://` OpenAPI URLs.
   - The lower generation layer should remain the single source of truth:
     - remote specs require `https`
     - loopback hosts may use `http`

## PR 2 Goals

Turn `kimbap service generate` into a practical onboarding path for real OpenAPI specs.

User-visible outcomes:

- loopback `http://localhost/...` OpenAPI URLs work
- `--name` overrides the generated service name
- repeatable `--tag` filters operations by OpenAPI tag
- repeatable `--path-prefix` filters operations by path prefix
- `--install` immediately installs the generated manifest
- generated action warnings are shown on `stderr` in text mode
- docs clearly present `service generate` as a supported workflow

Non-goals:

- no external `$ref` support yet
- no remote non-loopback `http://`
- no generic resolver rewrite
- no new manifest fields
- no update-from-generated-source persistence model

## UX Contract

### Source security

- `https://...` is allowed
- `http://localhost/...`, `http://127.0.0.1/...`, and `http://[::1]/...` are allowed
- remote non-loopback `http://...` is rejected
- redirects must not weaken the effective scheme policy

### `kimbap service generate`

Base behavior:

- without `--install`, the command generates YAML
- with `--output`, YAML is written to the target file
- without `--output`, YAML is printed to stdout

New flags:

- `--name <service-name>`
- repeatable `--tag <tag>`
- repeatable `--path-prefix <prefix>`
- `--install`

Filter semantics:

- multiple `--tag` flags are ORed together
- multiple `--path-prefix` flags are ORed together
- if both tag and path-prefix filters are present, an operation must satisfy both groups
- tag matching is case-insensitive
- path-prefix matching is prefix-based and should normalize a missing leading `/`
- if filters remove every operation, return a direct “no operations matched filters” error

Warnings:

- when generated actions contain warnings, text mode prints them to `stderr`
- JSON mode should stay quiet on `stderr` unless execution itself errors

`--install` behavior:

- install the generated manifest directly after generation
- if `--output` is also provided, write the file first, then install
- text mode prints a normal install success message instead of dumping YAML
- JSON mode should return a structured install payload instead of YAML

## Data And Code Design

Add a generation options layer in `internal/services`:

- `OpenAPIGenerateOptions`
  - `NameOverride string`
  - `Tags []string`
  - `PathPrefixes []string`

Preferred API shape:

- `GenerateFromOpenAPIWithOptions(spec []byte, opts OpenAPIGenerateOptions)`
- `GenerateFromOpenAPIFileWithOptions(path string, opts OpenAPIGenerateOptions)`
- `GenerateFromOpenAPIURLWithOptions(ctx, rawURL, opts)`

Keep current APIs as compatibility wrappers using zero-value options.

Why this is the right layer:

- filtering before action-key generation keeps naming deterministic
- name override belongs to manifest generation, not CLI patching
- loopback HTTP policy stays at the URL-generation layer

CLI design:

- `cmd/kimbap/service_generate.go` should:
  - parse flags
  - call the optioned generator
  - collect generated action warnings from the manifest
  - optionally install the manifest
  - render YAML or install output

## Implementation Steps

1. Add `OpenAPIGenerateOptions` and wrapper functions in `internal/services`.
2. Apply tag/path-prefix filtering inside action extraction before unique action key resolution.
3. Apply service-name override inside generation, followed by normal manifest validation.
4. Move `GenerateFromOpenAPIURL` logic to the optioned version and keep loopback HTTP support there.
5. Remove duplicate CLI-side rejection of all `http://` OpenAPI URLs.
6. Extend `service generate` flags and install flow.
7. Add warning collection/printing for generated actions.
8. Update docs in:
   - `README.md`
   - `docs/cli-reference.md`
   - `docs/service-development.md`

## Test Plan

### `internal/services`

1. loopback HTTP URL is accepted:
   - `localhost`
   - `127.0.0.1`
   - `::1`
2. remote HTTP URL is still rejected
3. tag filtering keeps only matching operations
4. path-prefix filtering keeps only matching operations
5. combined tag + path-prefix filtering uses AND semantics
6. name override changes the generated manifest name
7. filtering happens before action-key disambiguation

### `cmd/kimbap`

1. CLI no longer rejects loopback HTTP before reaching the service layer
2. `--name` affects generated YAML
3. repeatable `--tag` works
4. repeatable `--path-prefix` works
5. generated warnings print to `stderr` in text mode
6. `--install` installs generated manifests into the configured services dir
7. `--install --output` both writes the file and installs
8. remote insecure HTTP is still rejected with a clear message

## Review Checklist Before Coding

- Are the new filters implemented at generation time, not as CLI-only post-processing?
- Is loopback HTTP policy enforced in one place?
- Are existing public generator functions preserved as wrappers?
- Does `--install` avoid YAML-on-stdout in install mode?
- Are warnings emitted only in human/text mode?
- Are docs explicit about the localhost-HTTP exception?

## Review Checklist After Coding

- Run focused tests for `cmd/kimbap` and `internal/services`
- Verify loopback HTTP works from the CLI path
- Verify remote HTTP still fails
- Verify filtered generation keeps deterministic action keys
- Review diff for accidental scope creep into `$ref` support or manifest schema changes
