package services

type ServiceManifest struct {
	Name        string                   `yaml:"name"`
	Aliases     []string                 `yaml:"aliases,omitempty"`
	Version     string                   `yaml:"version"`
	Description string                   `yaml:"description"`
	Adapter     string                   `yaml:"adapter,omitempty"`
	CommandSpec *CommandSpec             `yaml:"command_spec,omitempty"`
	TargetApp   string                   `yaml:"target_app,omitempty"`
	BaseURL     string                   `yaml:"base_url"`
	Auth        ServiceAuth              `yaml:"auth"`
	Actions     map[string]ServiceAction `yaml:"actions"`
	Gotchas     []ServiceGotcha          `yaml:"gotchas,omitempty"`
	Triggers    *TriggerConfig           `yaml:"triggers,omitempty"`
	Recipes     []ServiceRecipe          `yaml:"recipes,omitempty"`
}

type ServiceAuth struct {
	Type          string `yaml:"type"`
	HeaderName    string `yaml:"header_name"`
	QueryParam    string `yaml:"query_param"`
	BodyField     string `yaml:"body_field"`
	CredentialRef string `yaml:"credential_ref"`
}

type CommandSpec struct {
	Executable string            `yaml:"executable"`
	JSONFlag   string            `yaml:"json_flag,omitempty"`
	Timeout    string            `yaml:"timeout,omitempty"`
	EnvInject  map[string]string `yaml:"env_inject,omitempty"`
}

type ServiceAction struct {
	Method       string         `yaml:"method"`
	Path         string         `yaml:"path"`
	Description  string         `yaml:"description"`
	Warnings     []string       `yaml:"warnings,omitempty"`
	Aliases      []string       `yaml:"aliases,omitempty"`
	Command      string         `yaml:"command,omitempty"`
	Idempotent   *bool          `yaml:"idempotent,omitempty"`
	Auth         *ServiceAuth   `yaml:"auth,omitempty"`
	Args         []ActionArg    `yaml:"args"`
	Request      RequestSpec    `yaml:"request"`
	Response     ResponseSpec   `yaml:"response"`
	Risk         RiskSpec       `yaml:"risk"`
	Retry        *RetrySpec     `yaml:"retry,omitempty"`
	Pagination   *PageSpec      `yaml:"pagination,omitempty"`
	ErrorMapping map[int]string `yaml:"error_mapping,omitempty"`
}

type ActionArg struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
	Default  any    `yaml:"default,omitempty"`
	Enum     []any  `yaml:"enum,omitempty"`
}

type RequestSpec struct {
	Query      map[string]string `yaml:"query,omitempty"`
	Headers    map[string]string `yaml:"headers,omitempty"`
	Body       map[string]any    `yaml:"body,omitempty"`
	PathParams map[string]string `yaml:"path_params,omitempty"`
}

type ResponseSpec struct {
	Extract string `yaml:"extract"`
	Type    string `yaml:"type"`
}

type RiskSpec struct {
	Level    string `yaml:"level"`
	Mutating *bool  `yaml:"mutating,omitempty"`
}

type RetrySpec struct {
	MaxAttempts int   `yaml:"max_attempts"`
	BackoffMS   int   `yaml:"backoff_ms"`
	RetryOn     []int `yaml:"retry_on"`
}

type PageSpec struct {
	Type     string `yaml:"type"`
	MaxPages int    `yaml:"max_pages"`
	NextPath string `yaml:"next_path"`
}

type ServiceGotcha struct {
	AppliesTo   string `yaml:"applies_to,omitempty"`
	Symptom     string `yaml:"symptom"`
	LikelyCause string `yaml:"likely_cause"`
	Recovery    string `yaml:"recovery"`
	Severity    string `yaml:"severity,omitempty"`
}

type TriggerConfig struct {
	TaskVerbs  []string `yaml:"task_verbs"`
	Objects    []string `yaml:"objects"`
	InsteadOf  []string `yaml:"instead_of,omitempty"`
	Exclusions []string `yaml:"exclusions,omitempty"`
}

type ServiceRecipe struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Steps       []string `yaml:"steps"`
}
