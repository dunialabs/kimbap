package services

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateSkillMD converts a ServiceManifest into the Agent Skills open standard
// SKILL.md format for cross-platform AI agent compatibility.
func GenerateSkillMD(manifest *ServiceManifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest is nil")
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", manifest.Name)
	desc := buildSkillDescription(manifest)
	fmt.Fprintf(&sb, "description: |\n  %s\n", strings.ReplaceAll(desc, "\n", "\n  "))
	sb.WriteString("allowed-tools: Bash\n")
	sb.WriteString("---\n\n")

	fmt.Fprintf(&sb, "# %s\n\n", manifest.Name)
	if manifest.Description != "" {
		fmt.Fprintf(&sb, "%s\n\n", manifest.Description)
	}

	sb.WriteString("## Prerequisites\n\n")
	sb.WriteString("- Kimbap CLI installed and in PATH\n")
	fmt.Fprintf(&sb, "- Service installed: `kimbap service install %s.yaml`\n", manifest.Name)
	credRefs := collectCredentialRefs(manifest)
	for _, ref := range credRefs {
		fmt.Fprintf(&sb, "- Credential configured: `kimbap vault set %s`\n", ref)
	}
	sb.WriteString("\n")

	sb.WriteString("## Available Actions\n\n")

	keys := sortedActionKeys(manifest.Actions)
	for _, key := range keys {
		action := manifest.Actions[key]
		actionName := manifest.Name + "." + key
		riskLevel := strings.ToLower(strings.TrimSpace(action.Risk.Level))
		riskDisplay := riskLevel
		if riskDisplay == "" {
			riskDisplay = "unknown"
		}
		fmt.Fprintf(&sb, "### %s\n\n", actionName)
		if action.Description != "" {
			fmt.Fprintf(&sb, "%s\n\n", action.Description)
		}
		if normalizedAdapterType(manifest.Adapter) == "applescript" {
			fmt.Fprintf(&sb, "**Command**: `%s`\n", action.Command)
		} else {
			fmt.Fprintf(&sb, "**HTTP**: `%s %s`\n", strings.ToUpper(action.Method), action.Path)
		}
		fmt.Fprintf(&sb, "**Risk**: %s\n\n", riskDisplay)

		if len(action.Args) > 0 {
			sb.WriteString("**Parameters**:\n")
			for _, arg := range action.Args {
				req := "optional"
				if arg.Required {
					req = "required"
				}
				fmt.Fprintf(&sb, "- `%s` (%s, %s)", arg.Name, arg.Type, req)
				if arg.Default != nil {
					fmt.Fprintf(&sb, " — default: `%v`", arg.Default)
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		sb.WriteString("**Usage**:\n")
		sb.WriteString("```bash\n")
		fmt.Fprintf(&sb, "kimbap call %s", actionName)
		for _, arg := range action.Args {
			if arg.Required {
				fmt.Fprintf(&sb, " --%s <value>", arg.Name)
			}
		}
		sb.WriteString("\n```\n")
		if riskLevel == "" || riskLevel == "unknown" || riskLevel == "medium" || riskLevel == "high" || riskLevel == "critical" {
			fmt.Fprintf(&sb, "\n> ⚠️ This action is risk level: %s. Use --dry-run --format json first to preview.\n", riskDisplay)
		}
		if riskLevel == "high" || riskLevel == "critical" {
			sb.WriteString("\n> 🔒 Approval may be required. Check: `kimbap approve list`\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Discovery\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List all actions from this service\n")
	fmt.Fprintf(&sb, "kimbap actions list --service %s --format json\n\n", manifest.Name)
	sb.WriteString("# Describe a specific action with full schema\n")
	fmt.Fprintf(&sb, "kimbap actions describe %s.<action> --format json\n\n", manifest.Name)
	sb.WriteString("# Dry-run to preview without executing\n")
	fmt.Fprintf(&sb, "kimbap call %s.<action> --dry-run --format json\n", manifest.Name)
	sb.WriteString("```\n")

	return sb.String(), nil
}

func buildSkillDescription(m *ServiceManifest) string {
	parts := []string{}
	if m.Description != "" {
		parts = append(parts, m.Description)
	}
	if normalizedAdapterType(m.Adapter) == "applescript" {
		parts = append(parts, fmt.Sprintf("Use when you need to control %s via AppleScript.", m.TargetApp))
	} else {
		parts = append(parts, fmt.Sprintf("Use when you need to interact with the %s API.", m.Name))
	}
	parts = append(parts, "Trigger phrases:")

	keys := sortedActionKeys(m.Actions)
	for _, key := range keys {
		action := m.Actions[key]
		if action.Description != "" {
			humanKey := strings.NewReplacer("_", " ", "-", " ").Replace(key)
			parts = append(parts, fmt.Sprintf("  - \"%s\": %s", humanKey, action.Description))
		}
	}

	return strings.Join(parts, "\n")
}

func sortedActionKeys(actions map[string]ServiceAction) []string {
	keys := make([]string, 0, len(actions))
	for k := range actions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func collectCredentialRefs(m *ServiceManifest) []string {
	seen := map[string]bool{}
	if normalizedAuthType(m.Auth.Type) != "none" && m.Auth.CredentialRef != "" {
		seen[m.Auth.CredentialRef] = true
	}
	for _, action := range m.Actions {
		if action.Auth != nil && normalizedAuthType(action.Auth.Type) != "none" && action.Auth.CredentialRef != "" {
			seen[action.Auth.CredentialRef] = true
		}
	}
	refs := make([]string, 0, len(seen))
	for ref := range seen {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	return refs
}

// metaSkillTemplate is the thin Tier-1 meta-skill that teaches AI agents
// how to connect to Kimbap and discover available actions at runtime.
// This content is stable and rarely changes — it does NOT list installed services.
const metaSkillTemplate = `---
name: kimbap
description: |
  Use Kimbap when the user needs to interact with external services
  (GitHub, Slack, Gmail, Stripe, Notion, internal APIs, etc.) through
  a secure, governed runtime. Kimbap provides credential injection,
  policy enforcement, approval workflows, and audit logging.
  Trigger phrases: 'List repositories', 'create issue', 'merge pull request',
  'send message', 'post to channel', 'send email', 'read inbox',
  'charge customer', 'list invoices', 'use external API',
  'call service', 'manage workspace', 'interact with third-party', 'use kimbap'.
allowed-tools: Bash
---

# Kimbap

> Secure action runtime for AI agents.
> Kimbap lets you use external services without handling raw credentials.

## Quick Start

` + "```bash" + `
# 1. Discover what services are available
kimbap actions list --format json

# 2. See all actions for a specific service
kimbap actions list --service <service-name> --format json

# 3. Inspect an action before using it
kimbap actions describe <service.action> --format json

# 4. Execute an action
kimbap call <service>.<action> [--param value ...]
` + "```" + `

## Rules

1. Always use ` + "`kimbap actions list --format json`" + ` first to discover what is available.
2. Use ` + "`kimbap actions describe <service.action> --format json`" + ` to inspect parameters before calling.
3. Never ask for, print, or store raw API keys, passwords, or tokens.
4. If a capability is missing, request a new Kimbap service instead of using direct credentials.
5. Treat Kimbap as the only approved pathway for third-party API access.

## Decision Protocol

Before calling any action:
1. ` + "`kimbap actions list --format json`" + `         # discover
2. ` + "`kimbap actions describe <service.action> --format json`" + `  # inspect schema
3. ` + "`kimbap call <service.action> --dry-run --format json`" + `    # preview
4. ` + "`kimbap call <service.action> [--params]`" + `       # execute
Never skip steps 1-3 for unfamiliar actions.

## Common Examples

` + "```bash" + `
# List all available actions
kimbap actions list --format json

# List installed services
kimbap service list

# Dry-run to preview without executing
kimbap call <service>.<action> --dry-run --format json

# Check what services are configured
kimbap actions list --format json
` + "```" + `

## Troubleshooting

` + "```bash" + `
Action not found:     kimbap service list
Auth failure:         kimbap connector list
Missing credential:   kimbap vault list
Approval required:    kimbap approve list
` + "```" + `

## Installation

` + "```bash" + `
# Install Kimbap CLI
# See https://kimbap.sh/quick-start

# Sync services to your AI agent
kimbap agents setup
` + "```" + `
`

// GenerateMetaSkillMD returns the Tier-1 meta-skill content.
func GenerateMetaSkillMD() string {
	return metaSkillTemplate
}

func GenerateSkillPack(manifest *ServiceManifest) (map[string]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}
	files := make(map[string]string)
	skillMD, err := generatePackSkillMD(manifest)
	if err != nil {
		return nil, fmt.Errorf("generate SKILL.md: %w", err)
	}
	files["SKILL.md"] = skillMD
	if g := generatePackGotchasMD(manifest); g != "" {
		files["GOTCHAS.md"] = g
	}
	if r := generatePackRecipesMD(manifest); r != "" {
		files["RECIPES.md"] = r
	}
	return files, nil
}

func generatePackSkillMD(manifest *ServiceManifest) (string, error) {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", manifest.Name))
	desc := buildPackDescription(manifest)
	sb.WriteString(fmt.Sprintf("description: |\n  %s\n", strings.ReplaceAll(desc, "\n", "\n  ")))
	sb.WriteString("allowed-tools: Bash\n")
	sb.WriteString("---\n\n")
	sb.WriteString(fmt.Sprintf("# %s\n\n", manifest.Name))
	if manifest.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", manifest.Description))
	}
	sb.WriteString("## Prerequisites\n\n")
	sb.WriteString("- Kimbap CLI installed and in PATH\n")
	sb.WriteString(fmt.Sprintf("- Service installed: `kimbap service install %s.yaml`\n", manifest.Name))
	for _, ref := range collectCredentialRefs(manifest) {
		sb.WriteString(fmt.Sprintf("- Credential configured: `kimbap vault set %s`\n", ref))
	}
	sb.WriteString("\n")
	sb.WriteString("## Available Actions\n\n")
	sb.WriteString("| Action | Description | Risk |\n")
	sb.WriteString("|--------|-------------|------|\n")
	for _, key := range sortedActionKeys(manifest.Actions) {
		action := manifest.Actions[key]
		risk := strings.ToLower(strings.TrimSpace(action.Risk.Level))
		if risk == "" {
			risk = "unknown"
		}
		d := action.Description
		if d == "" {
			d = "-"
		}
		d = strings.ReplaceAll(d, "\r\n", "\n")
		d = strings.ReplaceAll(d, "\n", "<br>")
		d = strings.ReplaceAll(d, "|", `\|`)
		risk = strings.ReplaceAll(risk, "|", `\|`)
		sb.WriteString(fmt.Sprintf("| `%s.%s` | %s | %s |\n", manifest.Name, key, d, risk))
	}
	sb.WriteString("\n")
	if len(manifest.Gotchas) > 0 {
		sb.WriteString("## Top Gotchas\n\n")
		limit := 3
		if len(manifest.Gotchas) < limit {
			limit = len(manifest.Gotchas)
		}
		for _, g := range manifest.Gotchas[:limit] {
			sb.WriteString(fmt.Sprintf("- **%s** → %s\n", g.Symptom, g.Recovery))
		}
		sb.WriteString("\n")
	}
	hasGotchas := len(manifest.Gotchas) > 0 || packHasActionWarnings(manifest)
	hasRecipes := len(manifest.Recipes) > 0
	if hasGotchas || hasRecipes {
		sb.WriteString("## Files in This Pack\n\n")
		if hasGotchas {
			sb.WriteString("- **GOTCHAS.md** — Common pitfalls, error patterns, and recovery steps\n")
		}
		if hasRecipes {
			sb.WriteString("- **RECIPES.md** — Multi-step workflow playbooks\n")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("## Discovery\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("kimbap actions list --service %s --format json\n", manifest.Name))
	sb.WriteString(fmt.Sprintf("kimbap actions describe %s.<action> --format json\n", manifest.Name))
	sb.WriteString("```\n")
	return sb.String(), nil
}

func buildPackDescription(manifest *ServiceManifest) string {
	if manifest.Triggers != nil {
		t := manifest.Triggers
		var parts []string
		if len(t.TaskVerbs) > 0 && len(t.Objects) > 0 {
			parts = append(parts, fmt.Sprintf("Use when you need to %s %s through approved Kimbap actions.",
				strings.Join(t.TaskVerbs, ", "),
				strings.Join(t.Objects, ", ")))
		}
		if len(t.InsteadOf) > 0 {
			parts = append(parts, fmt.Sprintf("Use instead of: %s.", strings.Join(t.InsteadOf, ", ")))
		}
		if len(t.Exclusions) > 0 {
			parts = append(parts, fmt.Sprintf("Do not use for: %s.", strings.Join(t.Exclusions, ", ")))
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
	}
	return buildSkillDescription(manifest)
}

func generatePackGotchasMD(manifest *ServiceManifest) string {
	if len(manifest.Gotchas) == 0 && !packHasActionWarnings(manifest) {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s — Common Pitfalls\n\n", manifest.Name))
	if len(manifest.Gotchas) > 0 {
		sb.WriteString("## Service-Level Gotchas\n\n")
		for _, g := range manifest.Gotchas {
			sb.WriteString(fmt.Sprintf("### %s\n\n", g.Symptom))
			sb.WriteString(fmt.Sprintf("**Likely cause**: %s\n\n", g.LikelyCause))
			sb.WriteString(fmt.Sprintf("**Recovery**: %s\n\n", g.Recovery))
			if g.Severity != "" {
				sb.WriteString(fmt.Sprintf("**Severity**: %s\n\n", g.Severity))
			}
		}
	}
	keys := sortedActionKeys(manifest.Actions)
	headerWritten := false
	for _, key := range keys {
		action := manifest.Actions[key]
		if len(action.Warnings) == 0 {
			continue
		}
		if !headerWritten {
			sb.WriteString("## Action-Specific Warnings\n\n")
			headerWritten = true
		}
		sb.WriteString(fmt.Sprintf("### %s.%s\n\n", manifest.Name, key))
		for _, w := range action.Warnings {
			sb.WriteString(fmt.Sprintf("- %s\n", w))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func generatePackRecipesMD(manifest *ServiceManifest) string {
	if len(manifest.Recipes) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s — Recipes\n\n", manifest.Name))
	for _, recipe := range manifest.Recipes {
		sb.WriteString(fmt.Sprintf("## %s\n\n", recipe.Name))
		if recipe.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", recipe.Description))
		}
		if len(recipe.Steps) > 0 {
			sb.WriteString("### Steps\n\n")
			for i, step := range recipe.Steps {
				sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func packHasActionWarnings(manifest *ServiceManifest) bool {
	for _, action := range manifest.Actions {
		if len(action.Warnings) > 0 {
			return true
		}
	}
	return false
}

func GenerateMetaSkillPack() map[string]string {
	return map[string]string{"SKILL.md": GenerateMetaSkillMD()}
}
