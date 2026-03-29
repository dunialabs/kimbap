package services

import (
	"fmt"
	"sort"
	"strings"
)

type SkillMDOption func(*skillMDConfig)

type skillMDConfig struct {
	Source string
}

func WithSource(source string) SkillMDOption {
	return func(c *skillMDConfig) {
		c.Source = source
	}
}

func buildInstallInstruction(name string, cfg skillMDConfig) string {
	source := strings.TrimSpace(cfg.Source)
	switch {
	case strings.HasPrefix(source, "official:"):
		return fmt.Sprintf("kimbap service install %s", name)
	case strings.HasPrefix(source, "remote:"):
		trimmed := strings.TrimSpace(strings.TrimPrefix(source, "remote:"))
		return fmt.Sprintf("kimbap service install %s", shellQuoteArg(trimmed))
	case strings.HasPrefix(source, "local:"):
		trimmed := strings.TrimSpace(strings.TrimPrefix(source, "local:"))
		return fmt.Sprintf("kimbap service install %s", shellQuoteArg(trimmed))
	case strings.HasPrefix(source, "github:"):
		withoutScheme := strings.TrimPrefix(source, "github:")
		if idx := strings.LastIndex(withoutScheme, ":"); idx >= 0 {
			installRef := "github:" + withoutScheme[:idx] + "/" + withoutScheme[idx+1:]
			return fmt.Sprintf("kimbap service install %s", shellQuoteArg(installRef))
		}
		return fmt.Sprintf("kimbap service install %s", shellQuoteArg(source))
	case source == "":
		return fmt.Sprintf("kimbap service install %s.yaml", name)
	default:
		return fmt.Sprintf("kimbap service install %s", name)
	}
}

// GenerateAgentSkillMD converts a ServiceManifest into the Agent Skills open standard
// SKILL.md format for cross-platform AI agent compatibility.
func GenerateAgentSkillMD(manifest *ServiceManifest, opts ...SkillMDOption) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("manifest is nil")
	}

	cfg := skillMDConfig{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}

	var sb strings.Builder

	sb.WriteString("---\n")
	fmt.Fprintf(&sb, "name: %s\n", manifest.Name)
	desc := buildAgentSkillDescription(manifest)
	fmt.Fprintf(&sb, "description: |\n  %s\n", strings.ReplaceAll(desc, "\n", "\n  "))
	sb.WriteString("allowed-tools: Bash\n")
	sb.WriteString("---\n\n")

	fmt.Fprintf(&sb, "# %s\n\n", manifest.Name)
	if manifest.Description != "" {
		fmt.Fprintf(&sb, "%s\n\n", manifest.Description)
	}

	sb.WriteString("## Prerequisites\n\n")
	sb.WriteString("- Kimbap CLI installed and in PATH\n")
	fmt.Fprintf(&sb, "- Service installed: `%s`\n", buildInstallInstruction(manifest.Name, cfg))
	credRefs := collectCredentialRefs(manifest)
	for _, ref := range credRefs {
		serviceName := ref
		if dot := strings.Index(ref, "."); dot > 0 {
			serviceName = ref[:dot]
		}
		if strings.TrimSpace(serviceName) == "" {
			serviceName = manifest.Name
		}
		fmt.Fprintf(&sb, "- Credential configured: `printf '%%s' \"$SECRET\" | kimbap link %s --stdin`\n", serviceName)
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
		switch normalizedAdapterType(manifest.Adapter) {
		case "applescript", "command":
			fmt.Fprintf(&sb, "**Command**: `%s`\n", action.Command)
		default:
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
		if action.Idempotent != nil && !*action.Idempotent {
			sb.WriteString("\n⚠️  Non-idempotent: pass --idempotency-key <unique-id> for safe retries.\n")
		}
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
	fmt.Fprintf(&sb, "kimbap actions list --service %s --format json\n", manifest.Name)
	fmt.Fprintf(&sb, "kimbap actions describe %s.<action> --format json\n", manifest.Name)
	fmt.Fprintf(&sb, "kimbap call %s.<action> --dry-run --format json\n", manifest.Name)
	sb.WriteString("```\n")

	return sb.String(), nil
}

func buildTriggerBasedDescription(t *TriggerConfig) string {
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
	return strings.Join(parts, "\n")
}

func buildAgentSkillDescription(m *ServiceManifest) string {
	if m.Triggers != nil {
		if desc := buildTriggerBasedDescription(m.Triggers); desc != "" {
			return desc
		}
	}
	parts := []string{}
	if m.Description != "" {
		parts = append(parts, m.Description)
	}
	switch normalizedAdapterType(m.Adapter) {
	case "applescript":
		parts = append(parts, fmt.Sprintf("Use when you need to control %s via AppleScript.", m.TargetApp))
	case "command":
		parts = append(parts, fmt.Sprintf("Use when you need to run %s commands.", m.Name))
	default:
		parts = append(parts, fmt.Sprintf("Use when you need to interact with the %s API.", m.Name))
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
1. Search by intent:    ` + "`kimbap search \"<intent>\" --format json`" + `
2. Inspect the action:  ` + "`kimbap actions describe <service.action> --format json`" + `
3. Preview if non-low:  ` + "`kimbap call <service.action> --dry-run --format json`" + `
4. Execute:             ` + "`kimbap call <service.action> [--param value ...]`" + `
Browse all services only when search returns nothing: ` + "`kimbap actions list --format json`" + `
</protocol>

<troubleshooting>
Action not found → ` + "`kimbap service list`" + ` | Auth failure → ` + "`kimbap auth list`" + ` | Missing credential → ` + "`kimbap vault list`" + ` | Approval required → ` + "`kimbap approve list`" + `
If no matching service exists, request a new Kimbap service instead of using direct credentials.
</troubleshooting>

<rules>
- If a service pack contains GOTCHAS.md or RECIPES.md, read them before unfamiliar or risky actions.
</rules>
`

// GenerateMetaAgentSkillMD returns the Tier-1 meta-skill content.
func GenerateMetaAgentSkillMD() string {
	return metaSkillTemplate
}

func GenerateAgentSkillPack(manifest *ServiceManifest, opts ...SkillMDOption) (map[string]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}
	cfg := skillMDConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	files := make(map[string]string)
	skillMD, err := generatePackSkillMD(manifest, cfg)
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

func generatePackSkillMD(manifest *ServiceManifest, _ skillMDConfig) (string, error) {
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
	sb.WriteString("## Available Actions\n\n")
	sb.WriteString("| Action | Description | Inputs | Risk |\n")
	sb.WriteString("|--------|-------------|--------|------|\n")
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
		if len(action.Warnings) > 0 {
			risk += " ⚠️"
		}
		risk = strings.ReplaceAll(risk, "|", `\|`)
		inputs := formatArgsSummary(action.Args)
		inputs = strings.ReplaceAll(inputs, "|", `\|`)
		sb.WriteString(fmt.Sprintf("| `%s.%s` | %s | %s | %s |\n", manifest.Name, key, d, inputs, risk))
	}
	sb.WriteString("\n")
	if len(manifest.Gotchas) > 0 {
		sb.WriteString("## Top Gotchas\n\n")
		topGotchas := topGotchasBySeverity(manifest.Gotchas, 3)
		for _, g := range topGotchas {
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
	if exampleSection := generateExampleCalls(manifest); exampleSection != "" {
		sb.WriteString(exampleSection)
		sb.WriteString("\n")
	}
	sb.WriteString("## Before Execute\n\n")
	fmt.Fprintf(&sb, "- Inspect: `kimbap actions describe %s.<action> --format json`\n", manifest.Name)
	fmt.Fprintf(&sb, "- Preview non-low-risk actions: `kimbap call %s.<action> --dry-run --format json`\n", manifest.Name)
	if hasGotchas {
		sb.WriteString("- Read GOTCHAS.md in this pack before unfamiliar or risky actions\n")
	}
	if hasRecipes {
		sb.WriteString("- Read RECIPES.md in this pack for workflow examples\n")
	}
	return sb.String(), nil
}

func buildPackDescription(manifest *ServiceManifest) string {
	if manifest.Triggers != nil {
		if desc := buildTriggerBasedDescription(manifest.Triggers); desc != "" {
			return desc
		}
	}
	return fmt.Sprintf("Use for approved %s actions through Kimbap.\nInspect the action table below for exact capabilities.", manifest.Name)
}

func generatePackGotchasMD(manifest *ServiceManifest) string {
	if len(manifest.Gotchas) == 0 && !packHasActionWarnings(manifest) {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s — Common Pitfalls\n\n", manifest.Name))
	if len(manifest.Gotchas) > 0 {
		sb.WriteString("## Service-Level Gotchas\n\n")
		for _, g := range topGotchasBySeverity(manifest.Gotchas, 0) {
			sb.WriteString(fmt.Sprintf("### %s\n\n", g.Symptom))
			if g.AppliesTo != "" {
				sb.WriteString(fmt.Sprintf("**Applies to**: `%s`\n\n", g.AppliesTo))
			}
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

func generateMetaAgentSkillPack() map[string]string {
	return map[string]string{"SKILL.md": GenerateMetaAgentSkillMD()}
}

func formatArgsSummary(args []ActionArg) string {
	if len(args) == 0 {
		return "-"
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		entry := "`" + arg.Name + "`"
		if !arg.Required {
			entry += "?"
		}
		parts = append(parts, entry)
	}
	if len(parts) > 4 {
		parts = append(parts[:3], "…")
	}
	return strings.Join(parts, ", ")
}

func severityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "common":
		return 3
	case "low":
		return 4
	case "rare":
		return 5
	default:
		return 6
	}
}

func topGotchasBySeverity(gotchas []ServiceGotcha, limit int) []ServiceGotcha {
	sorted := make([]ServiceGotcha, len(gotchas))
	copy(sorted, gotchas)
	sort.SliceStable(sorted, func(i, j int) bool {
		return severityRank(sorted[i].Severity) < severityRank(sorted[j].Severity)
	})
	if limit > 0 && len(sorted) > limit {
		sorted = sorted[:limit]
	}
	return sorted
}

func actionExamplePriority(action ServiceAction) int {
	risk := strings.ToLower(strings.TrimSpace(action.Risk.Level))
	switch risk {
	case "low":
		return 0
	case "medium":
		return 1
	case "high":
		return 2
	case "critical":
		return 3
	default:
		return 1
	}
}

func generateExampleCalls(manifest *ServiceManifest) string {
	keys := sortedActionKeys(manifest.Actions)
	if len(keys) == 0 {
		return ""
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return actionExamplePriority(manifest.Actions[keys[i]]) < actionExamplePriority(manifest.Actions[keys[j]])
	})

	var examples []string
	for _, key := range keys {
		action := manifest.Actions[key]
		var line strings.Builder
		line.WriteString(fmt.Sprintf("kimbap call %s.%s", manifest.Name, key))
		for _, arg := range action.Args {
			if arg.Required {
				line.WriteString(fmt.Sprintf(" --%s <value>", arg.Name))
			}
		}
		examples = append(examples, line.String())
		if len(examples) >= 2 {
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("## Example\n\n")
	sb.WriteString("```bash\n")
	for _, ex := range examples {
		sb.WriteString(ex + "\n")
	}
	sb.WriteString("```\n")
	return sb.String()
}

func shellQuoteArg(s string) string {
	const metaChars = " \t\n\"'\\`$&|;<>(){}!?*~#%"
	if !strings.ContainsAny(s, metaChars) {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
