package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ProfileType string

const (
	ProfileClaudeCode ProfileType = "claude-code"
	ProfileGeneric    ProfileType = "generic"
	ProfileCursor     ProfileType = "cursor"
)

type Profile struct {
	Name        ProfileType
	Template    string
	InstallPath string
}

const operatingRulesTemplate = `# Kimbap Operating Rules for Agents
1. Use Kimbap for external service access whenever possible.
2. Discover available actions with ` + "`kimbap actions list`" + `.
3. Inspect an action before using it with ` + "`kimbap actions describe <service.action>`" + `.
4. Execute via ` + "`kimbap call <service.action>`" + `.
5. For legacy apps or scripts, prefer ` + "`kimbap run`" + ` or a configured Kimbap proxy.
6. Never ask for, print, or store raw API keys, passwords, refresh tokens, cookies, or session files.
7. If the needed capability is missing, request a new Kimbap integration or skill instead of using direct credentials.
8. Treat Kimbap as the only approved pathway for third-party API access in this repository.
`

func GetProfile(name ProfileType) (*Profile, error) {
	switch name {
	case ProfileClaudeCode:
		return &Profile{
			Name:        ProfileClaudeCode,
			Template:    operatingRulesTemplate,
			InstallPath: filepath.Join(".claude", "KIMBAP_OPERATING_RULES.md"),
		}, nil
	case ProfileGeneric:
		return &Profile{
			Name:        ProfileGeneric,
			Template:    operatingRulesTemplate,
			InstallPath: filepath.Join(".agents", "KIMBAP_OPERATING_RULES.md"),
		}, nil
	case ProfileCursor:
		return &Profile{
			Name:        ProfileCursor,
			Template:    operatingRulesTemplate,
			InstallPath: filepath.Join(".cursor", "KIMBAP_OPERATING_RULES.md"),
		}, nil
	default:
		return nil, fmt.Errorf("unknown profile type: %q", name)
	}
}

func InstallProfile(profile *Profile, targetDir string) error {
	if profile == nil {
		return fmt.Errorf("profile is nil")
	}
	if strings.TrimSpace(targetDir) == "" {
		return fmt.Errorf("target directory is required")
	}

	fullPath := filepath.Join(targetDir, profile.InstallPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("create profile directory: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(profile.Template), 0o644); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	return nil
}

func PrintProfile(name ProfileType) (string, error) {
	profile, err := GetProfile(name)
	if err != nil {
		return "", err
	}
	return profile.Template, nil
}

type InstalledService struct {
	Name    string
	Actions []string
}

func GenerateDynamicProfile(name ProfileType, services []InstalledService) (*Profile, error) {
	base, err := GetProfile(name)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return base, nil
	}

	var sb strings.Builder
	sb.WriteString(base.Template)
	sb.WriteString("\n## Available Services\n\n")

	sorted := make([]InstalledService, len(services))
	copy(sorted, services)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	for _, svc := range sorted {
		fmt.Fprintf(&sb, "### %s\n", svc.Name)
		if len(svc.Actions) > 0 {
			actionsSorted := make([]string, len(svc.Actions))
			copy(actionsSorted, svc.Actions)
			sort.Strings(actionsSorted)
			sb.WriteString("```bash\n")
			for _, action := range actionsSorted {
				fmt.Fprintf(&sb, "kimbap call %s.%s\n", svc.Name, action)
			}
			sb.WriteString("```\n")
		}
		sb.WriteString("\n")
	}

	return &Profile{
		Name:        base.Name,
		Template:    sb.String(),
		InstallPath: base.InstallPath,
	}, nil
}
