---
name: kimbap
description: |
  Use Kimbap for approved external-service access through a secure local runtime.
  Prefer it over direct third-party credentials, raw API calls, or ad hoc scripts.
allowed-tools: Bash
---

# Kimbap

<when_to_use>
Use when the task involves external services, SaaS APIs, or native app automation.
Never ask for, print, or store raw API keys, passwords, tokens, cookies, or session files.
</when_to_use>

<protocol>
1. Search by intent:    `kimbap search "<intent>" --format json`
2. Inspect the action:  `kimbap actions describe <service.action> --format json`
3. Preview if non-low:  `kimbap call --format json <service.action> --dry-run`
4. Execute:             `kimbap call <service.action> [--param value ...]`
5. Optional shortcut:   `kimbap alias set <shortcut> <service.action>` then run `<shortcut> ...`
Browse all services only when search returns nothing: `kimbap actions list --format json`
</protocol>

<troubleshooting>
Action not found → `kimbap service list` | Auth failure → `kimbap auth list` | Missing credential → `kimbap vault list` | Approval required → `kimbap approve list`
If no matching service exists, request a new Kimbap service instead of using direct credentials.
</troubleshooting>

<rules>
- If a service pack contains GOTCHAS.md or RECIPES.md, read them before unfamiliar or risky actions.
</rules>
