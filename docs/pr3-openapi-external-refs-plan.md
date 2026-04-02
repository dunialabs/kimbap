# PR3 Plan: OpenAPI External `$ref` Support

## Recommendation

P3 still deserves a separate PR, but the right shape is a stacked PR on top of PR2 instead of a flat PR from `main`.

Why:

- PR2 and P3 both change the OpenAPI generation path and the same generator internals.
- A flat PR from `main` would either reintroduce already-reviewed generator churn or create a larger rebase/conflict step later.
- A stacked PR keeps the review focused on one capability change: relative file `$ref` resolution for local OpenAPI files.

Target base branch for the PR:

- `codex/pr2-service-generate-onboarding`

Target head branch for the PR:

- `codex/pr3-openapi-external-refs`

## Goal

Support realistic multi-file OpenAPI projects for local file generation by resolving relative file `$ref` chains during `GenerateFromOpenAPIFile`.

## Scope

In scope:

- Relative file `$ref` for `GenerateFromOpenAPIFile` and `GenerateFromOpenAPIFileWithOptions`
- Sibling and nested relative file resolution
- Mixed local and external refs within the same local file tree
- Clear errors for unsupported ref forms
- Tests covering multi-file YAML layouts
- Documentation note describing supported patterns

Out of scope:

- Remote URL ref trees
- Cross-origin ref fetching
- Arbitrary absolute-path ref support as a product feature
- Full JSON Schema / OpenAPI resolver parity

## Design

### 1. Keep byte-based generation semantics unchanged

`GenerateFromOpenAPI` and `GenerateFromOpenAPIWithOptions` should continue to support only in-memory specs plus local `#/...` refs. External file refs are only enabled for file-based generation because only that entrypoint has a safe base directory.

### 2. Make the resolver file-aware

Introduce file-backed document context inside `openAPIRefResolver`:

- track the root document for the current generation run
- cache loaded external documents by cleaned absolute path
- resolve relative file refs against the directory of the document that contains the `$ref`

### 3. Preserve document origin for nested maps

Nested external refs need to resolve relative to the file they came from, not always the original root file.

Implementation approach:

- register parsed maps with their source document
- when a map is later passed into `resolveMap`, recover its source document from resolver metadata
- when merging `$ref` targets with local sibling keys, register the merged map as belonging to the resolved target document while preserving existing child-map ownership metadata

This keeps nested refs working without rewriting the rest of the generator pipeline.

### 4. Support only relative file refs plus JSON Pointer fragments

Accepted patterns:

- `./schemas/pet.yaml#/Pet`
- `../common/params.yaml#/components/parameters/TraceId`
- `schemas/pet.yaml`
- `#/components/schemas/Pet`

Rejected patterns:

- `https://example.com/schema.yaml#/Foo`
- `http://example.com/schema.yaml#/Foo`
- `/absolute/path/schema.yaml#/Foo`
- non-pointer fragments such as `schema.yaml#Foo`

### 5. Keep errors explicit

Error messages should make the failure mode obvious:

- unsupported external ref form
- unable to read referenced file
- invalid referenced document
- unresolvable pointer inside referenced document

## Implementation Steps

1. Extend `openAPIRefResolver` with document cache and map-to-document tracking.
2. Split ref resolution into:
   - current-document local pointer resolution
   - relative file loading and pointer traversal
3. Update `GenerateFromOpenAPIFileWithOptions` to initialize a file-aware resolver rooted at the source file.
4. Keep `GenerateFromOpenAPIWithOptions` using a resolver with no external file support.
5. Add multi-file tests covering:
   - sibling file schema ref
   - nested external ref chain
   - external path-item / parameter / request-body usage
   - unsupported remote ref rejection
6. Document the supported external `$ref` patterns and the deliberate non-goals.

## Test Plan

Unit tests in `internal/services/generator_test.go`:

- generates from a split OpenAPI tree with relative schema refs
- resolves nested relative refs from a referenced file to another sibling file
- resolves external path items and component parameters used by actions
- rejects remote URL refs for local file generation
- rejects invalid external ref fragments

Regression coverage:

- existing single-file OpenAPI tests stay green
- existing URL generation tests stay green
- option filtering behavior from PR2 stays green

## Risks And Mitigations

Risk: map origin tracking becomes fragile.
Mitigation: track every parsed map recursively, register resolver-created merged maps explicitly, and keep the design scoped to resolver internals.

Risk: external ref support accidentally leaks into URL generation.
Mitigation: only enable file-relative resolution when the resolver is initialized from a local file path.

Risk: recursive refs cause runaway expansion.
Mitigation: keep the current bounded `resolveMap` loop and avoid eager deep inlining.

## Review Checklist

- No behavior change for byte-based or URL-based OpenAPI generation beyond file-entrypoint ref support
- Nested relative refs resolve from the referenced file's directory, not the root file's directory
- Unsupported remote refs fail early with explicit errors
- Tests cover both happy path and rejection path
- Docs describe supported patterns precisely
