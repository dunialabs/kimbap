package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/dunialabs/kimbap/internal/config"
	"github.com/dunialabs/kimbap/internal/registry"
	"github.com/dunialabs/kimbap/internal/services"
	"github.com/dunialabs/kimbap/services/catalog"
)

type catalogTriggerSummary struct {
	TaskVerbs  []string `json:"task_verbs,omitempty"`
	Objects    []string `json:"objects,omitempty"`
	InsteadOf  []string `json:"instead_of,omitempty"`
	Exclusions []string `json:"exclusions,omitempty"`
}

type catalogActionSummary struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	AuthType     string `json:"auth_type,omitempty"`
	AuthRequired bool   `json:"auth_required"`
}

type catalogGotchaSummary struct {
	Symptom  string `json:"symptom"`
	Severity string `json:"severity,omitempty"`
}

type catalogRecipeSummary struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type catalogServiceSummary struct {
	Name          string
	Description   string
	Aliases       []string
	Adapter       string
	AuthType      string
	CredentialRef string
	AuthRequired  bool
	Actions       []catalogActionSummary
	Triggers      *catalogTriggerSummary
	Gotchas       []catalogGotchaSummary
	Recipes       []catalogRecipeSummary
	Installed     bool
	Enabled       bool
	Status        string
	Shortcuts     []string
}

type catalogServiceListRow struct {
	Name         string                 `json:"name"`
	Catalog      bool                   `json:"catalog"`
	Installed    bool                   `json:"installed"`
	Enabled      bool                   `json:"enabled"`
	Status       string                 `json:"status"`
	Shortcuts    []string               `json:"shortcuts"`
	Description  string                 `json:"description"`
	Adapter      string                 `json:"adapter"`
	Actions      int                    `json:"actions"`
	AuthRequired bool                   `json:"auth_required"`
	Triggers     *catalogTriggerSummary `json:"triggers,omitempty"`
}

type catalogSearchResult struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Adapter        string   `json:"adapter"`
	AuthRequired   bool     `json:"auth_required"`
	Installed      bool     `json:"installed"`
	Enabled        bool     `json:"enabled"`
	Status         string   `json:"status"`
	Score          int      `json:"score"`
	MatchedActions []string `json:"matched_actions"`
	MatchedFields  []string `json:"matched_fields"`
}

type catalogDescribePayload struct {
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Aliases       []string               `json:"aliases,omitempty"`
	Adapter       string                 `json:"adapter"`
	AuthType      string                 `json:"auth_type"`
	CredentialRef string                 `json:"credential_ref,omitempty"`
	AuthRequired  bool                   `json:"auth_required"`
	Installed     bool                   `json:"installed"`
	Enabled       bool                   `json:"enabled"`
	Status        string                 `json:"status"`
	Shortcuts     []string               `json:"shortcuts"`
	Triggers      *catalogTriggerSummary `json:"triggers,omitempty"`
	ActionCount   int                    `json:"action_count"`
	Actions       []catalogActionSummary `json:"actions"`
	GotchaCount   int                    `json:"gotcha_count"`
	Gotchas       []catalogGotchaSummary `json:"gotchas"`
	RecipeCount   int                    `json:"recipe_count"`
	Recipes       []catalogRecipeSummary `json:"recipes"`
	InstallHint   string                 `json:"install_hint"`
}

func loadCatalogServiceSummaries(cfg *config.KimbapConfig) ([]catalogServiceSummary, error) {
	names, err := catalog.List()
	if err != nil {
		return nil, fmt.Errorf("list catalog services: %w", err)
	}

	installedByName := make(map[string]services.InstalledService)
	shortcutsByService := make(map[string][]string)
	if cfg != nil {
		shortcutsByService = shortcutsByServiceName(cfg.CommandAliases)
		installed, listErr := installerFromConfig(cfg).List()
		if listErr != nil {
			return nil, listErr
		}
		for _, svc := range installed {
			installedByName[svc.Manifest.Name] = svc
		}
	}

	summaries := make([]catalogServiceSummary, 0, len(names))
	for _, name := range names {
		data, getErr := catalog.Get(name)
		if getErr != nil {
			return nil, fmt.Errorf("load catalog service %q: %w", name, getErr)
		}
		manifest, parseErr := services.ParseManifest(data)
		if parseErr != nil {
			return nil, fmt.Errorf("parse catalog service %q: %w", name, parseErr)
		}

		summary := buildCatalogServiceSummary(manifest)
		summary.Shortcuts = cloneStringSlice(shortcutsByService[summary.Name])
		if summary.Shortcuts == nil {
			summary.Shortcuts = []string{}
		}
		if installed, ok := installedByName[summary.Name]; ok {
			summary.Installed = true
			summary.Enabled = installed.Enabled
			if installed.Enabled {
				summary.Status = "enabled"
			} else {
				summary.Status = "disabled"
			}
		} else {
			summary.Status = "not-installed"
		}

		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func buildCatalogServiceSummary(manifest *services.ServiceManifest) catalogServiceSummary {
	summary := catalogServiceSummary{
		Name:          strings.TrimSpace(manifest.Name),
		Description:   strings.TrimSpace(manifest.Description),
		Aliases:       cloneStringSlice(manifest.Aliases),
		Adapter:       catalogAdapterName(manifest),
		AuthType:      strings.TrimSpace(manifest.Auth.Type),
		CredentialRef: strings.TrimSpace(manifest.Auth.CredentialRef),
		AuthRequired:  serviceManifestRequiresCredentials(manifest),
		Actions:       make([]catalogActionSummary, 0, len(manifest.Actions)),
		Gotchas:       make([]catalogGotchaSummary, 0, len(manifest.Gotchas)),
		Recipes:       make([]catalogRecipeSummary, 0, len(manifest.Recipes)),
		Shortcuts:     []string{},
	}

	if triggerSummary := buildCatalogTriggerSummary(manifest.Triggers); triggerSummary != nil {
		summary.Triggers = triggerSummary
	}

	actionNames := make([]string, 0, len(manifest.Actions))
	for name := range manifest.Actions {
		actionNames = append(actionNames, name)
	}
	sort.Strings(actionNames)
	for _, name := range actionNames {
		action := manifest.Actions[name]
		authType := ""
		authRequired := false
		if action.Auth != nil {
			authType = strings.TrimSpace(action.Auth.Type)
			authRequired = authTypeRequiresCredential(authType)
		}
		summary.Actions = append(summary.Actions, catalogActionSummary{
			Name:         name,
			Description:  strings.TrimSpace(action.Description),
			AuthType:     authType,
			AuthRequired: authRequired,
		})
	}

	for _, gotcha := range manifest.Gotchas {
		summary.Gotchas = append(summary.Gotchas, catalogGotchaSummary{
			Symptom:  strings.TrimSpace(gotcha.Symptom),
			Severity: strings.TrimSpace(gotcha.Severity),
		})
	}

	for _, recipe := range manifest.Recipes {
		summary.Recipes = append(summary.Recipes, catalogRecipeSummary{
			Name:        strings.TrimSpace(recipe.Name),
			Description: strings.TrimSpace(recipe.Description),
		})
	}

	return summary
}

func buildCatalogTriggerSummary(triggers *services.TriggerConfig) *catalogTriggerSummary {
	if triggers == nil {
		return nil
	}

	summary := &catalogTriggerSummary{
		TaskVerbs:  cloneStringSlice(triggers.TaskVerbs),
		Objects:    cloneStringSlice(triggers.Objects),
		InsteadOf:  cloneStringSlice(triggers.InsteadOf),
		Exclusions: cloneStringSlice(triggers.Exclusions),
	}
	if len(summary.TaskVerbs) == 0 && len(summary.Objects) == 0 && len(summary.InsteadOf) == 0 && len(summary.Exclusions) == 0 {
		return nil
	}
	return summary
}

func catalogListRows(summaries []catalogServiceSummary) []catalogServiceListRow {
	rows := make([]catalogServiceListRow, 0, len(summaries))
	for _, summary := range summaries {
		rows = append(rows, catalogServiceListRow{
			Name:         summary.Name,
			Catalog:      true,
			Installed:    summary.Installed,
			Enabled:      summary.Enabled,
			Status:       summary.Status,
			Shortcuts:    cloneStringSlice(summary.Shortcuts),
			Description:  summary.Description,
			Adapter:      summary.Adapter,
			Actions:      len(summary.Actions),
			AuthRequired: summary.AuthRequired,
			Triggers:     cloneCatalogTriggerSummary(summary.Triggers),
		})
	}
	return rows
}

func buildCatalogDescribePayload(summary catalogServiceSummary) catalogDescribePayload {
	return catalogDescribePayload{
		Name:          summary.Name,
		Description:   summary.Description,
		Aliases:       cloneStringSlice(summary.Aliases),
		Adapter:       summary.Adapter,
		AuthType:      summary.AuthType,
		CredentialRef: summary.CredentialRef,
		AuthRequired:  summary.AuthRequired,
		Installed:     summary.Installed,
		Enabled:       summary.Enabled,
		Status:        summary.Status,
		Shortcuts:     cloneStringSlice(summary.Shortcuts),
		Triggers:      cloneCatalogTriggerSummary(summary.Triggers),
		ActionCount:   len(summary.Actions),
		Actions:       cloneCatalogActionSummaries(summary.Actions),
		GotchaCount:   len(summary.Gotchas),
		Gotchas:       cloneCatalogGotchaSummaries(summary.Gotchas),
		RecipeCount:   len(summary.Recipes),
		Recipes:       cloneCatalogRecipeSummaries(summary.Recipes),
		InstallHint:   catalogServiceInstallHint(summary),
	}
}

func findCatalogServiceSummary(summaries []catalogServiceSummary, name string) (catalogServiceSummary, bool) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	for _, summary := range summaries {
		if strings.EqualFold(summary.Name, normalized) {
			return summary, true
		}
	}
	return catalogServiceSummary{}, false
}

func catalogServiceNotFoundError(name string, summaries []catalogServiceSummary) error {
	trimmed := strings.TrimSpace(name)
	candidates := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		candidates = append(candidates, summary.Name)
	}
	hint := "Run 'kimbap service list --available' to see all catalog services."
	if suggestion := didYouMean(trimmed, candidates); suggestion != "" {
		hint = fmt.Sprintf("Did you mean %q? Run 'kimbap service list --available' to see all catalog services.", suggestion)
	}
	return fmt.Errorf("%w. %s", &registry.ErrNotFound{Name: trimmed, Registry: "catalog"}, hint)
}

func scoreCatalogService(summary catalogServiceSummary, terms []string) (int, []string, []string) {
	if len(terms) == 0 {
		return 0, nil, nil
	}

	score := 0
	matchedFields := make([]string, 0, 6)
	matchedActions := make([]string, 0, 4)
	addField := func(field string) {
		if !slices.Contains(matchedFields, field) {
			matchedFields = append(matchedFields, field)
		}
	}
	addAction := func(name string) {
		if !slices.Contains(matchedActions, name) {
			matchedActions = append(matchedActions, name)
		}
	}

	name := strings.ToLower(summary.Name)
	description := strings.ToLower(summary.Description)
	adapter := strings.ToLower(summary.Adapter)
	aliasValues := lowercaseStrings(summary.Aliases)

	for _, term := range terms {
		if strings.Contains(name, term) {
			score += 8
			addField("name")
		}
		if containsString(aliasValues, term) {
			score += 5
			addField("aliases")
		}
		if strings.Contains(description, term) {
			score += 4
			addField("description")
		}
		if strings.Contains(adapter, term) {
			score += 2
			addField("adapter")
		}
		if catalogTriggerMatches(summary.Triggers, term) {
			score += 3
			addField("triggers")
		}
		for _, action := range summary.Actions {
			actionName := strings.ToLower(action.Name)
			actionDescription := strings.ToLower(action.Description)
			if strings.Contains(actionName, term) {
				score += 4
				addField("action_names")
				addAction(action.Name)
			}
			if strings.Contains(actionDescription, term) {
				score += 3
				addField("action_descriptions")
				addAction(action.Name)
			}
		}
		for _, recipe := range summary.Recipes {
			if strings.Contains(strings.ToLower(recipe.Name), term) || strings.Contains(strings.ToLower(recipe.Description), term) {
				score += 2
				addField("recipes")
			}
		}
	}

	sort.Strings(matchedFields)
	sort.Strings(matchedActions)
	return score, matchedFields, matchedActions
}

func catalogTriggerMatches(triggers *catalogTriggerSummary, term string) bool {
	if triggers == nil {
		return false
	}
	for _, values := range [][]string{triggers.TaskVerbs, triggers.Objects, triggers.InsteadOf, triggers.Exclusions} {
		if containsString(lowercaseStrings(values), term) {
			return true
		}
	}
	return false
}

func catalogSearchResults(summaries []catalogServiceSummary, terms []string) []catalogSearchResult {
	results := make([]catalogSearchResult, 0)
	for _, summary := range summaries {
		score, matchedFields, matchedActions := scoreCatalogService(summary, terms)
		if score <= 0 {
			continue
		}
		results = append(results, catalogSearchResult{
			Name:           summary.Name,
			Description:    summary.Description,
			Adapter:        summary.Adapter,
			AuthRequired:   summary.AuthRequired,
			Installed:      summary.Installed,
			Enabled:        summary.Enabled,
			Status:         summary.Status,
			Score:          score,
			MatchedActions: matchedActions,
			MatchedFields:  matchedFields,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			return results[i].Name < results[j].Name
		}
		return results[i].Score > results[j].Score
	})

	return results
}

func catalogAdapterName(manifest *services.ServiceManifest) string {
	if manifest == nil {
		return ""
	}
	adapter := strings.ToLower(strings.TrimSpace(manifest.Adapter))
	if adapter == "" {
		return "http"
	}
	return adapter
}

func catalogServiceInstallHint(summary catalogServiceSummary) string {
	switch {
	case !summary.Installed:
		return fmt.Sprintf("Run 'kimbap service install %s' to install this service.", summary.Name)
	case summary.Enabled:
		return fmt.Sprintf("Service %q is already installed and enabled.", summary.Name)
	default:
		return fmt.Sprintf("Run 'kimbap service enable %s' to enable the installed service.", summary.Name)
	}
}

func cloneCatalogActionSummaries(items []catalogActionSummary) []catalogActionSummary {
	if items == nil {
		return nil
	}
	out := make([]catalogActionSummary, len(items))
	copy(out, items)
	return out
}

func cloneCatalogGotchaSummaries(items []catalogGotchaSummary) []catalogGotchaSummary {
	if items == nil {
		return nil
	}
	out := make([]catalogGotchaSummary, len(items))
	copy(out, items)
	return out
}

func cloneCatalogRecipeSummaries(items []catalogRecipeSummary) []catalogRecipeSummary {
	if items == nil {
		return nil
	}
	out := make([]catalogRecipeSummary, len(items))
	copy(out, items)
	return out
}

func cloneCatalogTriggerSummary(summary *catalogTriggerSummary) *catalogTriggerSummary {
	if summary == nil {
		return nil
	}
	return &catalogTriggerSummary{
		TaskVerbs:  cloneStringSlice(summary.TaskVerbs),
		Objects:    cloneStringSlice(summary.Objects),
		InsteadOf:  cloneStringSlice(summary.InsteadOf),
		Exclusions: cloneStringSlice(summary.Exclusions),
	}
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func lowercaseStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, strings.ToLower(strings.TrimSpace(value)))
	}
	return out
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
