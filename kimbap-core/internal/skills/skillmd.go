package skills

import (
	"fmt"
	"sort"
	"strings"
)

// GenerateSkillMD converts a SkillManifest into the Agent Skills open standard
// SKILL.md format for cross-platform AI agent compatibility.
func GenerateSkillMD(manifest *SkillManifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest is nil")
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", manifest.Name))
	desc := buildSkillDescription(manifest)
	sb.WriteString(fmt.Sprintf("description: |\n  %s\n", strings.ReplaceAll(desc, "\n", "\n  ")))
	sb.WriteString("allowed-tools: Bash\n")
	sb.WriteString("---\n\n")

	sb.WriteString(fmt.Sprintf("# %s\n\n", manifest.Name))
	if manifest.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", manifest.Description))
	}

	sb.WriteString("## Prerequisites\n\n")
	sb.WriteString("- Kimbap CLI installed and in PATH\n")
	sb.WriteString(fmt.Sprintf("- Skill installed: `kimbap skill install %s.yaml`\n", manifest.Name))
	if manifest.Auth.Type != "none" {
		sb.WriteString(fmt.Sprintf("- Credential configured: `kimbap vault set %s`\n", manifest.Auth.CredentialRef))
	}
	sb.WriteString("\n")

	sb.WriteString("## Available Actions\n\n")

	keys := sortedActionKeys(manifest.Actions)
	for _, key := range keys {
		action := manifest.Actions[key]
		actionName := manifest.Name + "." + key
		sb.WriteString(fmt.Sprintf("### %s\n\n", actionName))
		if action.Description != "" {
			sb.WriteString(fmt.Sprintf("%s\n\n", action.Description))
		}
		sb.WriteString(fmt.Sprintf("**HTTP**: `%s %s`\n", strings.ToUpper(action.Method), action.Path))
		sb.WriteString(fmt.Sprintf("**Risk**: %s\n\n", action.Risk.Level))

		if len(action.Args) > 0 {
			sb.WriteString("**Parameters**:\n")
			for _, arg := range action.Args {
				req := "optional"
				if arg.Required {
					req = "required"
				}
				sb.WriteString(fmt.Sprintf("- `%s` (%s, %s)", arg.Name, arg.Type, req))
				if arg.Default != nil {
					sb.WriteString(fmt.Sprintf(" — default: `%v`", arg.Default))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}

		sb.WriteString("**Usage**:\n")
		sb.WriteString("```bash\n")
		sb.WriteString(fmt.Sprintf("kimbap call %s", actionName))
		for _, arg := range action.Args {
			if arg.Required {
				sb.WriteString(fmt.Sprintf(" --%s <value>", arg.Name))
			}
		}
		sb.WriteString("\n```\n\n")
	}

	sb.WriteString("## Discovery\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List all actions from this skill\n")
	sb.WriteString(fmt.Sprintf("kimbap actions list --service %s --format json\n\n", manifest.Name))
	sb.WriteString("# Describe a specific action with full schema\n")
	sb.WriteString(fmt.Sprintf("kimbap actions describe %s.<action> --format json\n\n", manifest.Name))
	sb.WriteString("# Dry-run to preview without executing\n")
	sb.WriteString(fmt.Sprintf("kimbap call %s.<action> --dry-run --format json\n", manifest.Name))
	sb.WriteString("```\n")

	return sb.String(), nil
}

func buildSkillDescription(m *SkillManifest) string {
	parts := []string{}
	if m.Description != "" {
		parts = append(parts, m.Description)
	}
	parts = append(parts, fmt.Sprintf("Use when you need to interact with the %s API.", m.Name))
	parts = append(parts, "Trigger phrases:")

	keys := sortedActionKeys(m.Actions)
	for _, key := range keys {
		action := m.Actions[key]
		if action.Description != "" {
			parts = append(parts, fmt.Sprintf("  - \"%s\": %s", key, action.Description))
		}
	}

	return strings.Join(parts, "\n")
}

func sortedActionKeys(actions map[string]SkillAction) []string {
	keys := make([]string, 0, len(actions))
	for k := range actions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GenerateMetaSkillMD produces the thin Tier-1 meta-skill that teaches AI agents
// how to connect to Kimbap and discover available actions at runtime.
// This skill is stable and rarely changes — it does NOT list installed services.
func GenerateMetaSkillMD() string {
	var sb strings.Builder

	sb.WriteString("---\n")
	sb.WriteString("name: kimbap\n")
	sb.WriteString("description: |\n")
	sb.WriteString("  Use Kimbap when the user needs to interact with external services\n")
	sb.WriteString("  (GitHub, Slack, Gmail, Stripe, Notion, internal APIs, etc.) through\n")
	sb.WriteString("  a secure, governed runtime. Kimbap provides credential injection,\n")
	sb.WriteString("  policy enforcement, approval workflows, and audit logging.\n")
	sb.WriteString("  Trigger phrases: 'use GitHub', 'send email', 'create issue',\n")
	sb.WriteString("  'call external API', 'interact with service', 'use kimbap'.\n")
	sb.WriteString("allowed-tools: Bash\n")
	sb.WriteString("---\n\n")

	sb.WriteString("# Kimbap\n\n")
	sb.WriteString("> Secure action runtime for AI agents.\n")
	sb.WriteString("> Kimbap lets you use external services without handling raw credentials.\n\n")

	sb.WriteString("## Quick Start\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# 1. Discover what services are available\n")
	sb.WriteString("kimbap actions list\n\n")
	sb.WriteString("# 2. See all actions for a specific service\n")
	sb.WriteString("kimbap actions list --service <service-name>\n\n")
	sb.WriteString("# 3. Inspect an action before using it\n")
	sb.WriteString("kimbap actions describe <service.action>\n\n")
	sb.WriteString("# 4. Execute an action\n")
	sb.WriteString("kimbap call <service>.<action> [--param value ...]\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## Rules\n\n")
	sb.WriteString("1. Always use `kimbap actions list` first to discover what is available.\n")
	sb.WriteString("2. Use `kimbap actions describe <service.action>` to inspect parameters before calling.\n")
	sb.WriteString("3. Never ask for, print, or store raw API keys, passwords, or tokens.\n")
	sb.WriteString("4. If a capability is missing, request a new Kimbap skill instead of using direct credentials.\n")
	sb.WriteString("5. Treat Kimbap as the only approved pathway for third-party API access.\n\n")

	sb.WriteString("## Common Examples\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# List all available actions\n")
	sb.WriteString("kimbap actions list\n\n")
	sb.WriteString("# List installed skills\n")
	sb.WriteString("kimbap skill list\n\n")
	sb.WriteString("# Dry-run to preview without executing\n")
	sb.WriteString("kimbap call <service>.<action> --dry-run\n\n")
	sb.WriteString("# Check what services are configured\n")
	sb.WriteString("kimbap actions list --format json\n")
	sb.WriteString("```\n\n")

	sb.WriteString("## Installation\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Install Kimbap CLI\n")
	sb.WriteString("# See https://kimbap.sh/quick-start\n\n")
	sb.WriteString("# Sync skills to your AI agent\n")
	sb.WriteString("kimbap agents setup\n")
	sb.WriteString("```\n")

	return sb.String()
}
