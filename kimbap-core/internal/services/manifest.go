package services

type ServiceManifest struct {
	Name        string                   `yaml:"name"`
	Version     string                   `yaml:"version"`
	Description string                   `yaml:"description"`
	Adapter     string                   `yaml:"adapter,omitempty"`
	TargetApp   string                   `yaml:"target_app,omitempty"`
	BaseURL     string                   `yaml:"base_url"`
	Auth        ServiceAuth              `yaml:"auth"`
	Actions     map[string]ServiceAction `yaml:"actions"`
}

type ServiceAuth struct {
	Type          string `yaml:"type"`
	HeaderName    string `yaml:"header_name"`
	QueryParam    string `yaml:"query_param"`
	BodyField     string `yaml:"body_field"`
	CredentialRef string `yaml:"credential_ref"`
}

type ServiceAction struct {
	Method       string         `yaml:"method"`
	Path         string         `yaml:"path"`
	Description  string         `yaml:"description"`
	Warnings     []string       `yaml:"warnings,omitempty"`
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
	Level string `yaml:"level"`
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
