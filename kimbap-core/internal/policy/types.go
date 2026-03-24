package policy

type PolicyDocument struct {
	Version string       `yaml:"version"`
	Rules   []PolicyRule `yaml:"rules"`
}

type PolicyRule struct {
	ID          string            `yaml:"id"`
	Description string            `yaml:"description"`
	Priority    int               `yaml:"priority"`
	Match       PolicyMatch       `yaml:"match"`
	Decision    PolicyDecision    `yaml:"decision"`
	Conditions  []PolicyCondition `yaml:"conditions,omitempty"`
	RateLimit   *RateLimitRule    `yaml:"rate_limit,omitempty"`
	TimeWindow  *TimeWindow       `yaml:"time_window,omitempty"`
}

type PolicyMatch struct {
	Agents   []string `yaml:"agents,omitempty"`
	Services []string `yaml:"services,omitempty"`
	Actions  []string `yaml:"actions,omitempty"`
	Risk     []string `yaml:"risk,omitempty"`
	Tenants  []string `yaml:"tenants,omitempty"`
}

type PolicyDecision string

const (
	DecisionAllow           PolicyDecision = "allow"
	DecisionDeny            PolicyDecision = "deny"
	DecisionRequireApproval PolicyDecision = "require_approval"
)

type PolicyCondition struct {
	Field    string `yaml:"field"`
	Operator string `yaml:"operator"`
	Value    any    `yaml:"value"`
}

type RateLimitRule struct {
	MaxRequests int    `yaml:"max_requests"`
	WindowSec   int    `yaml:"window_sec"`
	Scope       string `yaml:"scope"`
}

type TimeWindow struct {
	After    string   `yaml:"after"`
	Before   string   `yaml:"before"`
	Timezone string   `yaml:"timezone,omitempty"`
	Weekdays []string `yaml:"weekdays,omitempty"`
}
