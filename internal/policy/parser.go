package policy

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	semverLikePattern = regexp.MustCompile(`^v?\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)
	decisionSet       = map[PolicyDecision]struct{}{
		DecisionAllow:           {},
		DecisionDeny:            {},
		DecisionRequireApproval: {},
	}
	conditionOperatorSet = map[string]struct{}{
		"eq": {}, "ne": {}, "gt": {}, "lt": {}, "gte": {}, "lte": {}, "in": {}, "contains": {},
	}
	rateLimitScopeSet = map[string]struct{}{"agent": {}, "tenant": {}, "action": {}}
)

type ValidationError struct {
	Field   string
	Message string
}

func ParseDocument(data []byte) (*PolicyDocument, error) {
	var doc PolicyDocument
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse policy document: %w", err)
	}
	errs := ValidateDocument(&doc)
	if len(errs) > 0 {
		return nil, validationErrorsToError("policy document validation failed", errs)
	}
	return &doc, nil
}

func ParseDocumentFile(path string) (*PolicyDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy file: %w", err)
	}
	return ParseDocument(data)
}

func ValidateDocument(doc *PolicyDocument) []ValidationError {
	if doc == nil {
		return []ValidationError{{Field: "document", Message: "document is required"}}
	}

	errs := make([]ValidationError, 0)

	if !semverLikePattern.MatchString(doc.Version) {
		errs = append(errs, ValidationError{Field: "version", Message: "must be semver-like (e.g. 1.2.3 or v1.2.3)"})
	}

	if len(doc.Rules) == 0 {
		errs = append(errs, ValidationError{Field: "rules", Message: "must contain at least one rule"})
		return errs
	}

	seenIDs := make(map[string]struct{}, len(doc.Rules))
	for idx, rule := range doc.Rules {
		prefix := fmt.Sprintf("rules[%d]", idx)

		if strings.TrimSpace(rule.ID) == "" {
			errs = append(errs, ValidationError{Field: prefix + ".id", Message: "is required"})
		} else {
			if _, exists := seenIDs[rule.ID]; exists {
				errs = append(errs, ValidationError{Field: prefix + ".id", Message: "must be unique"})
			}
			seenIDs[rule.ID] = struct{}{}
		}

		if _, ok := decisionSet[rule.Decision]; !ok {
			errs = append(errs, ValidationError{Field: prefix + ".decision", Message: "must be allow, deny, or require_approval"})
		}

		for cidx, cond := range rule.Conditions {
			cp := fmt.Sprintf("%s.conditions[%d]", prefix, cidx)
			if strings.TrimSpace(cond.Field) == "" {
				errs = append(errs, ValidationError{Field: cp + ".field", Message: "is required"})
			}
			if _, ok := conditionOperatorSet[strings.ToLower(strings.TrimSpace(cond.Operator))]; !ok {
				errs = append(errs, ValidationError{Field: cp + ".operator", Message: "must be eq, ne, gt, lt, gte, lte, in, contains"})
			}
		}

		if rule.RateLimit != nil {
			if rule.RateLimit.MaxRequests <= 0 {
				errs = append(errs, ValidationError{Field: prefix + ".rate_limit.max_requests", Message: "must be > 0"})
			}
			if rule.RateLimit.WindowSec <= 0 {
				errs = append(errs, ValidationError{Field: prefix + ".rate_limit.window_sec", Message: "must be > 0"})
			}
			if _, ok := rateLimitScopeSet[strings.ToLower(strings.TrimSpace(rule.RateLimit.Scope))]; !ok {
				errs = append(errs, ValidationError{Field: prefix + ".rate_limit.scope", Message: "must be agent, tenant, or action"})
			}
		}

		if rule.TimeWindow != nil {
			errs = append(errs, validateTimeWindow(prefix+".time_window", rule.TimeWindow)...)
		}

		for _, mf := range []struct {
			fieldName string
			patterns  []string
		}{
			{"match.agents", rule.Match.Agents},
			{"match.services", rule.Match.Services},
			{"match.actions", rule.Match.Actions},
			{"match.risk", rule.Match.Risk},
			{"match.tenants", rule.Match.Tenants},
		} {
			for pidx, pattern := range mf.patterns {
				if _, err := path.Match(pattern, ""); err != nil {
					errs = append(errs, ValidationError{
						Field:   fmt.Sprintf("%s.%s[%d]", prefix, mf.fieldName, pidx),
						Message: fmt.Sprintf("invalid glob pattern %q: %v", pattern, err),
					})
				}
			}
		}
	}

	return errs
}

var validWeekdaySet = map[string]struct{}{
	"monday": {}, "tuesday": {}, "wednesday": {}, "thursday": {},
	"friday": {}, "saturday": {}, "sunday": {},
	"mon": {}, "tue": {}, "wed": {}, "thu": {}, "fri": {}, "sat": {}, "sun": {},
}

func validateTimeWindow(prefix string, tw *TimeWindow) []ValidationError {
	var errs []ValidationError
	if after := strings.TrimSpace(tw.After); after != "" {
		if _, err := time.Parse("15:04", after); err != nil {
			errs = append(errs, ValidationError{Field: prefix + ".after", Message: "must be HH:MM format (e.g. 09:00)"})
		}
	}
	if before := strings.TrimSpace(tw.Before); before != "" {
		if _, err := time.Parse("15:04", before); err != nil {
			errs = append(errs, ValidationError{Field: prefix + ".before", Message: "must be HH:MM format (e.g. 17:00)"})
		}
	}
	if tz := strings.TrimSpace(tw.Timezone); tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			errs = append(errs, ValidationError{Field: prefix + ".timezone", Message: "must be a valid IANA timezone (e.g. America/New_York)"})
		}
	}
	for i, wd := range tw.Weekdays {
		if _, ok := validWeekdaySet[strings.ToLower(strings.TrimSpace(wd))]; !ok {
			errs = append(errs, ValidationError{Field: fmt.Sprintf("%s.weekdays[%d]", prefix, i), Message: "must be a valid weekday name or abbreviation"})
		}
	}
	return errs
}

func validationErrorsToError(prefix string, errs []ValidationError) error {
	parts := make([]string, 0, len(errs))
	for _, err := range errs {
		parts = append(parts, fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return fmt.Errorf("%s: %s", prefix, strings.Join(parts, "; "))
}
