package actions

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"
)

type RiskLevel string

const (
	RiskRead        RiskLevel = "read"
	RiskWrite       RiskLevel = "write"
	RiskAdmin       RiskLevel = "admin"
	RiskDestructive RiskLevel = "destructive"
)

func (r RiskLevel) DocVocab() string {
	switch r {
	case RiskRead:
		return "low"
	case RiskWrite:
		return "medium"
	case RiskAdmin:
		return "high"
	case RiskDestructive:
		return "critical"
	default:
		return string(r)
	}
}

type ApprovalHint string

const (
	ApprovalNone     ApprovalHint = "none"
	ApprovalOptional ApprovalHint = "optional"
	ApprovalRequired ApprovalHint = "required"
)

type ExecutionMode string

const (
	ModeCall  ExecutionMode = "call"
	ModeRun   ExecutionMode = "run"
	ModeProxy ExecutionMode = "proxy"
	ModeServe ExecutionMode = "serve"
)

type ExecutionStatus string

const (
	StatusSuccess          ExecutionStatus = "success"
	StatusError            ExecutionStatus = "error"
	StatusApprovalRequired ExecutionStatus = "approval_required"
	StatusTimeout          ExecutionStatus = "timeout"
	StatusCancelled        ExecutionStatus = "cancelled"
)

type Principal struct {
	ID        string
	TenantID  string
	AgentName string
	Type      string
	Scopes    []string
}

type SessionContext struct {
	SessionID string
	Mode      ExecutionMode
	Channel   string
	SourceIP  string
	UserAgent string
	StartedAt time.Time
	Meta      map[string]any
}

type ResolvedCredentialSet struct {
	Type      string
	Token     string
	APIKey    string
	Username  string
	Password  string
	Headers   map[string]string
	Query     map[string]string
	Body      map[string]any
	ExpiresAt *time.Time
	Meta      map[string]any
}

type ClassificationInfo struct {
	Service       string
	ActionName    string
	Method        string
	Path          string
	Host          string
	MatchedRuleID string
	Confidence    float64
}

type ApprovalContext struct {
	Required    bool
	ApproverIDs []string
	Reason      string
	TicketRef   string
	Deadline    *time.Time
	Meta        map[string]any
}

type AuthType string

const (
	AuthTypeNone    AuthType = "none"
	AuthTypeBearer  AuthType = "bearer"
	AuthTypeAPIKey  AuthType = "api_key"
	AuthTypeBasic   AuthType = "basic"
	AuthTypeHeader  AuthType = "header"
	AuthTypeQuery   AuthType = "query"
	AuthTypeBody    AuthType = "body"
	AuthTypeOAuth2  AuthType = "oauth2"
	AuthTypeSession AuthType = "session"
)

type AuthRequirement struct {
	Type          AuthType
	CredentialRef string
	HeaderName    string
	QueryName     string
	BodyField     string
	Prefix        string
	Optional      bool
	Scopes        []string
	Audience      string
}

type AdapterConfig struct {
	Type           string
	Method         string
	BaseURL        string
	ExecutablePath string
	URLTemplate    string
	Headers        map[string]string
	Query          map[string]string
	EnvInject      map[string]string
	JSONFlag       string
	RequestBody    string
	Response       ResponseConfig
	Retry          RetryConfig
	Timeout        time.Duration
	AllowInsecure  bool
	TargetApp      string
	Command        string
	ScriptSource   string
	ScriptLanguage string
	ApprovalRef    string
	AuditRef       string
	RegistryMode   string
}

type ResponseConfig struct {
	Extract     string
	ErrorPath   string
	StatusPath  string
	HeadersPath string
}

type RetryConfig struct {
	MaxAttempts int
	BackoffMS   int
	RetryOn429  bool
	RetryOn5xx  bool
}

type Schema struct {
	Type                 string
	Required             []string
	Properties           map[string]*Schema
	Items                *Schema
	Enum                 []any
	AdditionalProperties bool
}

type PaginationConfig struct {
	Style          string
	LimitParam     string
	CursorParam    string
	OffsetParam    string
	DefaultLimit   int
	MaxPages       int
	ResponseCursor string
}

type RateLimitConfig struct {
	Scope         string
	Requests      int
	Per           time.Duration
	Burst         int
	RetryAfterMS  int
	ProviderLimit string
}

type ClassifierRule struct {
	Service     string
	Action      string
	Method      string
	PathPattern string
	HostPattern string
}

type ActionDefinition struct {
	Name         string
	Version      int
	DisplayName  string
	Namespace    string
	Verb         string
	Resource     string
	Description  string
	Risk         RiskLevel
	Idempotent   bool
	ApprovalHint ApprovalHint
	Auth         AuthRequirement
	InputSchema  *Schema
	OutputSchema *Schema
	Defaults     map[string]any
	Adapter      AdapterConfig
	Classifiers  []ClassifierRule
	ErrorMapping map[int]string
	Pagination   *PaginationConfig
	RateLimit    *RateLimitConfig
	// Output filtering configuration applied at the action level (adapter-agnostic)
	FilterConfig *FilterConfig // adapter-agnostic output filter
	// Compact text template for list/search results
	CompactTemplate *CompactTemplate // compact text template for list/search
}

type ExecutionRequest struct {
	RequestID       string
	TraceID         string
	TenantID        string
	Principal       Principal
	Action          ActionDefinition
	Input           map[string]any
	Mode            ExecutionMode
	Session         *SessionContext
	Credentials     *ResolvedCredentialSet
	Classification  *ClassificationInfo
	ApprovalContext *ApprovalContext
	IdempotencyKey  string
	Timeout         time.Duration
}

type ExecutionResult struct {
	RequestID      string
	TraceID        string
	Status         ExecutionStatus
	Output         map[string]any
	HTTPStatus     int
	Error          *ExecutionError
	Retryable      bool
	IdempotencyKey string
	DurationMS     int64
	PolicyDecision string // allow, deny, require_approval
	AuditRef       string
	Meta           map[string]any
}

type ResultEnvelope struct {
	Data map[string]any `json:"data,omitempty"`
	Meta map[string]any `json:"_meta,omitempty"`
}

func ValidateInput(schema *Schema, input map[string]any) *ExecutionError {
	if schema == nil {
		return nil
	}
	if schema.Type != "" && schema.Type != "object" {
		return NewExecutionError(
			ErrValidationFailed,
			"root schema must be object",
			400,
			false,
			map[string]any{"type": schema.Type},
		)
	}
	if input == nil {
		input = map[string]any{}
	}
	if schema.Type == "" && schema.Properties == nil && len(schema.Required) == 0 {
		return nil
	}

	for _, key := range schema.Required {
		value, ok := input[key]
		if !ok || requiredValueMissing(schema.Properties[key], value) {
			return NewExecutionError(
				ErrValidationFailed,
				fmt.Sprintf("missing required field %q", key),
				400,
				false,
				map[string]any{"field": key},
			)
		}
	}

	for key, value := range input {
		if schema.Properties == nil {
			if !schema.AdditionalProperties {
				return NewExecutionError(
					ErrValidationFailed,
					fmt.Sprintf("unknown field %q", key),
					400,
					false,
					map[string]any{"field": key},
				)
			}
			continue
		}

		prop, ok := schema.Properties[key]
		if !ok {
			if !schema.AdditionalProperties {
				return NewExecutionError(
					ErrValidationFailed,
					fmt.Sprintf("unknown field %q", key),
					400,
					false,
					map[string]any{"field": key},
				)
			}
			continue
		}
		if err := validateValue(key, prop, value); err != nil {
			return err
		}
	}

	return nil
}

// FilterConfig holds adapter-agnostic output filtering configuration for an action.
type FilterConfig struct {
	Select    map[string]string
	Exclude   []string
	MaxItems  int
	DropNulls bool
}

// FilterMeta tracks metadata about applied filtering for an action's output.
type FilterMeta struct {
	Applied            bool
	FieldsSelected     int
	FieldsExcluded     int
	ItemsTruncatedFrom int
	OriginalBytes      int
	FilteredBytes      int
	PartialMiss        []string
	Skipped            string
}

// CompactTemplate defines a template for compact textual representations of items.
type CompactTemplate struct {
	Header string
	Item   string
	Footer string
}

func requiredValueMissing(fieldSchema *Schema, value any) bool {
	if fieldSchema == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(fieldSchema.Type), "string") {
		return false
	}
	v, ok := value.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(v) == ""
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func numericEqual(a, b any) bool {
	ai, aInt := toInt64(a)
	bi, bInt := toInt64(b)
	if aInt && bInt {
		return ai == bi
	}
	fa, aOK := toFloat64(a)
	fb, bOK := toFloat64(b)
	return aOK && bOK && fa == fb
}

func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int8:
		return int64(n), true
	case int16:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case uint:
		if n <= math.MaxInt64 {
			return int64(n), true
		}
	case uint8:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint64:
		if n <= math.MaxInt64 {
			return int64(n), true
		}
	case float64:
		if n == math.Trunc(n) && n >= math.MinInt64 && n <= math.MaxInt64 {
			return int64(n), true
		}
	}
	return 0, false
}

func validateValue(field string, schema *Schema, value any) *ExecutionError {
	if schema == nil {
		return nil
	}
	if len(schema.Enum) > 0 {
		matched := false
		for _, candidate := range schema.Enum {
			if reflect.DeepEqual(candidate, value) {
				matched = true
				break
			}
			if numericEqual(candidate, value) {
				matched = true
				break
			}
		}
		if !matched {
			return NewExecutionError(
				ErrValidationFailed,
				fmt.Sprintf("field %q has invalid enum value", field),
				400,
				false,
				map[string]any{"field": field},
			)
		}
	}

	typeName := strings.ToLower(schema.Type)
	if typeName == "" {
		return nil
	}

	valid := false
	switch typeName {
	case "string":
		_, valid = value.(string)
	case "number":
		switch value.(type) {
		case float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			valid = true
		}
	case "integer":
		switch v := value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			valid = true
		case float64:
			valid = math.Trunc(v) == v
		}
	case "boolean":
		_, valid = value.(bool)
	case "object":
		var obj map[string]any
		obj, valid = value.(map[string]any)
		if valid {
			for _, key := range schema.Required {
				if _, ok := obj[key]; !ok {
					return NewExecutionError(
						ErrValidationFailed,
						fmt.Sprintf("field %q missing required nested field %q", field, key),
						400,
						false,
						map[string]any{"field": field, "nested_field": key},
					)
				}
			}
			if !schema.AdditionalProperties {
				for key := range obj {
					if _, declared := schema.Properties[key]; !declared {
						return NewExecutionError(
							ErrValidationFailed,
							fmt.Sprintf("field %q has unknown nested field %q", field, key),
							400,
							false,
							map[string]any{"field": field, "nested_field": key},
						)
					}
				}
			}
			for key, prop := range schema.Properties {
				nested, ok := obj[key]
				if !ok {
					continue
				}
				if err := validateValue(fmt.Sprintf("%s.%s", field, key), prop, nested); err != nil {
					return err
				}
			}
		}
	case "array":
		arr, ok := value.([]any)
		if ok {
			valid = true
			if schema.Items != nil {
				for idx, item := range arr {
					if err := validateValue(fmt.Sprintf("%s[%d]", field, idx), schema.Items, item); err != nil {
						return err
					}
				}
			}
		}
	}

	if !valid {
		return NewExecutionError(
			ErrValidationFailed,
			fmt.Sprintf("field %q must be %s", field, typeName),
			400,
			false,
			map[string]any{"field": field, "expected": typeName},
		)
	}

	return nil
}
