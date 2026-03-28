# Service Development Guide

Kimbap services are defined as YAML manifests. No code is needed for most integrations. This guide covers the manifest format, all three adapter types, validation, and contributing.

---

## Quick Start

The fastest path to a working service:

1. Write a minimal YAML manifest (see examples below).
2. Validate it.
3. Install it.
4. Call an action.

```bash
kimbap service validate my-service.yaml
kimbap service install my-service.yaml
kimbap call my-service.my-action
```

A minimal HTTP manifest looks like this:

```yaml
name: my-service
version: 1.0.0
base_url: https://api.example.com
auth:
  type: none
actions:
  ping:
    method: GET
    path: /ping
    description: Health check
    risk:
      level: low
    idempotent: true
```

That's enough to install and call. Everything else in this guide is optional enrichment.

---

## Adapter Types

Three adapters are available. Omitting the `adapter` field defaults to `http`.

| Adapter | Use case | Key field |
|---|---|---|
| `http` | REST APIs, SaaS integrations | `base_url` |
| `command` | CLI tool wrappers | `command_spec` |
| `applescript` | macOS native applications | `target_app` |

---

## HTTP Adapter

The HTTP adapter sends requests to a remote base URL. Each action maps to a method and path.

### Full Example

```yaml
name: github
version: 1.0.0
description: GitHub REST API integration for repository listing and issue management
base_url: https://api.github.com
auth:
  type: bearer
  credential_ref: github.token
triggers:
  task_verbs: [list, inspect, create, fetch]
  objects: [repositories, issues]
  instead_of:
    - calling GitHub APIs directly with raw tokens
  exclusions:
    - unrelated local file operations
gotchas:
  - symptom: 401/403 on endpoints that should be accessible
    likely_cause: Token lacks required scopes or is tied to the wrong account/installation
    recovery: Confirm token type and scopes and re-auth with correct principal
    severity: high
recipes:
  - name: Create issue from investigation notes
    description: Convert findings into a tracked GitHub issue with consistent metadata
    steps:
      - List repositories to confirm the target owner/repo
      - Create an issue with title/body and optional labels
      - Fetch the created issue by issue_number to verify persisted content
actions:
  list-repos:
    method: GET
    path: /user/repos
    description: List repositories for the authenticated user
    args:
      - name: sort
        type: string
        required: false
        default: updated
        enum: [created, updated, pushed, full_name]
      - name: per_page
        type: integer
        required: false
        default: 30
    request:
      query:
        sort: "{sort}"
        per_page: "{per_page}"
      headers:
        Accept: application/vnd.github+json
    response:
      type: array
    risk:
      level: low
    idempotent: true
  create-issue:
    method: POST
    path: /repos/{owner}/{repo}/issues
    description: Create a new issue in a GitHub repository
    args:
      - name: owner
        type: string
        required: true
      - name: repo
        type: string
        required: true
      - name: title
        type: string
        required: true
      - name: body
        type: string
        required: false
      - name: labels
        type: array
        required: false
    request:
      path_params:
        owner: "{owner}"
        repo: "{repo}"
      headers:
        Accept: application/vnd.github+json
      body:
        title: "{title}"
        body: "{body}"
        labels: "{labels}"
    response:
      type: object
    risk:
      level: medium
    idempotent: false
```

### Key Concepts

**`base_url`** must be an absolute `http://` or `https://` URL. It must not include a query string or fragment. Correct: `https://api.github.com`. Incorrect: `https://api.github.com?version=2022`.

**`path`** is appended to `base_url` and must start with `/`. Path parameters like `{owner}` are interpolated from the `path_params` map in the request block.

**`request` block** controls how args map into the outgoing request. Values like `"{sort}"` are string templates referencing arg names. You can place arg values into `query`, `headers`, `body`, or `path_params`.

**`auth.credential_ref`** is a dot-separated path into the kimbap vault. `github.token` refers to the `token` key under the `github` namespace. Set it with `printf '%s' "$TOKEN" | kimbap link github --stdin` or directly via `kimbap vault set github.token --stdin`.

---

## Command Adapter

The command adapter runs a local executable for each action. It's suited for CLI tools that accept structured arguments and return JSON.

### Full Example

```yaml
name: blender
version: 1.0.0
description: 3D scene creation, modeling, and rendering via CLI-Anything Blender harness
adapter: command
command_spec:
  executable: cli-anything-blender
  json_flag: "--json"
  timeout: "300s"
auth:
  type: none
actions:
  create-scene:
    command: "scene new"
    description: Initialize a new Blender scene project
    idempotent: false
    args:
      - name: output
        type: string
        required: true
    response:
      type: object
    risk:
      level: medium
  render:
    command: "render execute"
    description: Render a scene project to an output image
    idempotent: false
    args:
      - name: project
        type: string
        required: true
      - name: output
        type: string
        required: true
      - name: resolution_x
        type: integer
        required: false
        default: 1920
      - name: resolution_y
        type: integer
        required: false
        default: 1080
      - name: engine
        type: string
        required: false
        default: EEVEE
    response:
      type: object
    risk:
      level: high
```

### Key Concepts

**`command_spec`** is required for the command adapter and has four fields:

- `executable` (required): the binary name or full path. It must be on `$PATH` or an absolute path.
- `json_flag` (optional): a flag appended to every invocation to request JSON output (e.g. `--json`, `--output json`).
- `timeout` (optional): a duration string like `"30s"` or `"5m"`. Defaults to the global runtime timeout if omitted.
- `env_inject` (optional): a map of environment variables to inject at invocation time. Values are passed through literally.

**`command`** on each action is the subcommand string passed to the executable. For the example above, `kimbap call blender.render` would run something equivalent to `cli-anything-blender render execute --json --project ... --output ...`.

Actions under the command adapter do not have `method` or `path` fields. The `args` block still works the same way.

---

## AppleScript Adapter

The AppleScript adapter automates macOS applications by dispatching named handlers. It's macOS-only.

### Full Example

```yaml
name: finder
version: 1.0.0
description: Finder automation via AppleScript — file system operations on macOS
adapter: applescript
target_app: Finder
auth:
  type: none
actions:
  list-items:
    command: finder-list-items
    description: List files and folders in a directory
    idempotent: true
    args:
      - name: path
        type: string
        required: true
    response:
      type: array
    risk:
      level: low
  move-item:
    command: finder-move-item
    description: Move a file or folder to another directory
    idempotent: false
    args:
      - name: source_path
        type: string
        required: true
      - name: destination_path
        type: string
        required: true
    response:
      type: object
    risk:
      level: high
```

### Key Concepts

**`target_app`** is required for the applescript adapter. It must be the exact name of the macOS application as it appears in Script Editor (e.g. `Finder`, `Safari`, `Mail`).

**`command`** on each action must reference a registered AppleScript command handler. Kimbap dispatches to this handler by name, passing args as a record. If the handler is not registered, the action will fail at runtime with a handler-not-found error.

The adapter does not use `method`, `path`, `base_url`, or `command_spec`. Auth type must be `none` for AppleScript actions.

---

## Complete Schema Reference

### ServiceManifest (top-level fields)

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Must match `[a-z][a-z0-9-]*` |
| `version` | string | yes | Semver-like, e.g. `1.0.0` or `v1.2.3` |
| `description` | string | no | Free text, shown in listings and SKILL.md |
| `adapter` | string | no | `http`, `command`, or `applescript`. Defaults to `http` |
| `base_url` | string | required for http | Absolute `http`/`https` URL, no query string or fragment |
| `command_spec` | object | required for command | See sub-fields below |
| `target_app` | string | required for applescript | Exact macOS application name |
| `auth` | object | yes | See sub-fields below |
| `triggers` | object | no | Used for agent discovery and SKILL.md export |
| `gotchas` | array | no | Known failure patterns and recovery advice |
| `recipes` | array | no | Suggested multi-step workflows |
| `actions` | map | yes | At least one action required |

### command_spec fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `executable` | string | yes | Binary name or absolute path |
| `json_flag` | string | no | Flag appended to every invocation to request JSON |
| `timeout` | string | no | Duration string, e.g. `"300s"`, `"5m"` |
| `env_inject` | map | no | Environment variables to inject. Values are passed through literally |

### auth fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `type` | string | yes | `header`, `bearer`, `basic`, `query`, `body`, or `none` |
| `header_name` | string | for `header` type | The HTTP header to set |
| `query_param` | string | for `query` type | The query parameter name |
| `body_field` | string | for `body` type | The request body field name |
| `credential_ref` | string | when type requires a credential | Dot-separated vault key path, e.g. `github.token` |

### triggers fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `task_verbs` | string[] | yes, if triggers present | At least one verb required |
| `objects` | string[] | yes, if triggers present | At least one object required |
| `instead_of` | string[] | no | Describes what this service replaces |
| `exclusions` | string[] | no | Describes what this service should not be used for |

### gotchas entry fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `symptom` | string | yes | Observable failure or unexpected behavior |
| `likely_cause` | string | yes | Root cause or explanation |
| `recovery` | string | yes | Steps or advice to resolve |
| `severity` | string | no | `low`, `medium`, `high`, `critical`, `common`, or `rare` |
| `applies_to` | string | no | Specific action or context this applies to |

### recipes entry fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Short name for the recipe |
| `description` | string | no | What the recipe accomplishes |
| `steps` | string[] | yes | At least one step required |

### ServiceAction (each entry in the `actions` map)

| Field | Type | Required | Notes |
|---|---|---|---|
| action key | string | yes | Must match `[a-z][a-z0-9_-]*` |
| `method` | string | for http | `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, or `OPTIONS` |
| `path` | string | for http | Must start with `/` |
| `command` | string | for command/applescript | Subcommand string or handler name |
| `description` | string | no | Human-readable description |
| `idempotent` | boolean | no | Whether repeated calls have the same effect |
| `warnings` | string[] | no | Free-text warnings shown before execution |
| `auth` | object | no | Per-action auth override; same shape as top-level auth |
| `args` | array | no | Input argument definitions |
| `request` | object | no | Controls how args map to the outgoing request (HTTP only) |
| `response` | object | no | Describes the expected response shape |
| `risk` | object | yes | Risk classification |
| `retry` | object | no | Retry configuration |
| `pagination` | object | no | Pagination configuration |
| `error_mapping` | map | no | HTTP status codes mapped to error messages |

### args entry fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Unique within the action. Used in template strings |
| `type` | string | yes | `string`, `integer`, `number`, `boolean`, `array`, or `object` |
| `required` | boolean | yes | If true, caller must provide a value |
| `default` | any | no | Must not be set on required args. Type must match declared type |
| `enum` | array | no | Restricts accepted values to this list |

### request fields (HTTP adapter)

| Field | Type | Notes |
|---|---|---|
| `query` | map | Query string parameters. Values are template strings |
| `headers` | map | Request headers. Values are template strings |
| `body` | map | JSON request body fields. Values are template strings |
| `path_params` | map | Path segment substitutions. Values are template strings |

### response fields

| Field | Type | Notes |
|---|---|---|
| `type` | string | `object` or `array` |
| `extract` | string | Dot-path expression (with optional `[index]`) to pull a sub-value from the response |

### risk fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `level` | string | yes | `low`, `medium`, `high`, or `critical` |

### retry fields

| Field | Type | Notes |
|---|---|---|
| `max_attempts` | int | Total attempts including the first |
| `backoff_ms` | int | Milliseconds to wait between attempts |
| `retry_on` | int[] | HTTP status codes that trigger a retry |

### pagination fields

| Field | Type | Notes |
|---|---|---|
| `type` | string | `cursor` or `offset` |
| `max_pages` | int | Hard limit on pages fetched |
| `next_path` | string | JSONPath to the next cursor or offset in the response |

### error_mapping

A map from HTTP status code (as a string key, e.g. `"404"`) to a human-readable error message. Used to surface cleaner errors to the caller.

---

## Validation Rules

`kimbap service validate` checks the following. Any failure produces a specific error message pointing to the offending field.

**Name and version**
- `name` must match `[a-z][a-z0-9-]*`. No uppercase, no underscores at the top level.
- `version` must be semver-like: `1.0.0`, `v1.2.3`, `0.1.0`, etc.

**Actions**
- At least one action must be defined.
- Each action key must match `[a-z][a-z0-9_-]*`. Underscores are allowed in action keys but not the service name.

**Risk**
- `risk.level` must be one of `low`, `medium`, `high`, or `critical`.

**Args**
- Arg names must be unique within a single action.
- Arg `type` must be one of `string`, `integer`, `number`, `boolean`, `array`, or `object`.
- Required args (`required: true`) must not have a `default` value.
- If a `default` is set, its Go type must match the declared `type` (e.g. a default of `30` on a `string` arg will fail).

**HTTP adapter**
- `base_url` must be set and must be an absolute URL with scheme `http` or `https`.
- `base_url` must not include a query string (`?`) or fragment (`#`).
- `method` must be one of `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD`, or `OPTIONS`.
- `path` must start with `/`.

**Command adapter**
- `command_spec` must be present.
- `command_spec.executable` is required and must not be empty.

**AppleScript adapter**
- `target_app` must be set and must not be empty.
- Each action's `command` must reference a registered AppleScript handler name.

**Triggers**
- If `triggers` is present, both `task_verbs` and `objects` must have at least one entry each.

**Gotchas**
- If any gotcha entry is present, all three of `symptom`, `likely_cause`, and `recovery` are required on that entry.

**Recipes**
- Each recipe must have at least one entry in `steps`.

---

## Testing Workflow

```bash
# Validate the manifest before installing
kimbap service validate my-service.yaml

# Install from a local file
kimbap service install my-service.yaml

# Call an action and pass arguments
kimbap call my-service.my-action --arg value

# Pass multiple args
kimbap call github.create-issue --owner myorg --repo myrepo --title "Bug: something broke"

# Generate SKILL.md for agent discovery
kimbap service export-agent-skill my-service
```

To check what's installed:

```bash
kimbap service list
```

To remove an installed service:

```bash
kimbap service remove my-service
```

---

## Contributing a Service

1. Write a YAML manifest following this guide.
2. Run `kimbap service validate my-service.yaml` and fix all errors.
3. Install locally with `kimbap service install my-service.yaml` and test at least one action with `kimbap call`.
4. Place the manifest in `skills/official/` using the service name as the filename (e.g. `skills/official/github.yaml`).
5. Open a pull request. The PR description should note which adapter type is used, what credentials are required, and what you tested.
6. See `CONTRIBUTING.md` for code style standards, review process, and required fields for official services.

Official services are expected to include `triggers`, at least one `recipe`, and at least one `gotcha` entry where relevant failure modes exist.
