package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Proxy struct {
	ID              int     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name            string  `gorm:"column:name;type:varchar(128);not null" json:"name"`
	Addtime         int     `gorm:"column:addtime;not null" json:"addtime"`
	ProxyKey        string  `gorm:"column:proxy_key;type:varchar(255);not null;default:''" json:"proxyKey"`
	StartPort       int     `gorm:"column:start_port;not null;default:3002" json:"startPort"`
	LogWebhookURL   *string `gorm:"column:log_webhook_url;type:text" json:"logWebhookUrl"`
	LastSyncedLogID int     `gorm:"column:last_synced_log_id;not null;default:0" json:"lastSyncedLogId"`
}

func (Proxy) TableName() string { return "proxy" }

type User struct {
	UserID          string  `gorm:"column:user_id;primaryKey;type:varchar(64);not null" json:"userId"`
	Status          int     `gorm:"column:status;not null" json:"status"`
	Role            int     `gorm:"column:role;not null" json:"role"`
	Permissions     string  `gorm:"column:permissions;not null" json:"permissions"`
	UserPreferences string  `gorm:"column:user_preferences;not null;default:''" json:"userPreferences"`
	LaunchConfigs   string  `gorm:"column:launch_configs;not null;default:'{}'" json:"launchConfigs"`
	ExpiresAt       int     `gorm:"column:expires_at;not null;default:0" json:"expiresAt"`
	CreatedAt       int     `gorm:"column:created_at;not null;default:0" json:"createdAt"`
	UpdatedAt       int     `gorm:"column:updated_at;not null;default:0" json:"updatedAt"`
	Ratelimit       int     `gorm:"column:ratelimit;not null" json:"ratelimit"`
	Name            string  `gorm:"column:name;type:varchar(128);not null" json:"name"`
	EncryptedToken  *string `gorm:"column:encrypted_token" json:"encryptedToken"`
	ProxyID         int     `gorm:"column:proxy_id;not null;default:0" json:"proxyId"`
	Notes           *string `gorm:"column:notes" json:"notes"`
}

func (User) TableName() string { return "user" }

type Server struct {
	ServerID                string  `gorm:"column:server_id;primaryKey;type:varchar(128);not null" json:"serverId"`
	ServerName              string  `gorm:"column:server_name;type:varchar(128);not null" json:"serverName"`
	Enabled                 bool    `gorm:"column:enabled;not null;default:true" json:"enabled"`
	LaunchConfig            string  `gorm:"column:launch_config;not null" json:"launchConfig"`
	Capabilities            string  `gorm:"column:capabilities;not null" json:"capabilities"`
	CreatedAt               int     `gorm:"column:created_at;not null;default:0" json:"createdAt"`
	UpdatedAt               int     `gorm:"column:updated_at;not null;default:0" json:"updatedAt"`
	AllowUserInput          bool    `gorm:"column:allow_user_input;not null;default:false" json:"allowUserInput"`
	ConfigTemplate          *string `gorm:"column:config_template;type:text;default:''" json:"configTemplate"`
	ProxyID                 int     `gorm:"column:proxy_id;not null;default:0" json:"proxyId"`
	ToolTmplID              *string `gorm:"column:tool_tmpl_id;type:varchar(128)" json:"toolTmplId"`
	AuthType                int     `gorm:"column:auth_type;not null;default:1" json:"authType"`
	Category                int     `gorm:"column:category;not null;default:1" json:"category"`
	PublicAccess            bool    `gorm:"column:public_access;not null;default:false" json:"publicAccess"`
	UseKimbapOauthConfig      bool    `gorm:"column:use_kimbap_oauth_config;not null;default:true" json:"useKimbapOauthConfig"`
	TransportType           *string `gorm:"column:transport_type;type:varchar(10)" json:"transportType"`
	LazyStartEnabled        bool    `gorm:"column:lazy_start_enabled;not null;default:true" json:"lazyStartEnabled"`
	CachedTools             *string `gorm:"column:cached_tools;type:text" json:"cachedTools"`
	CachedResources         *string `gorm:"column:cached_resources;type:text" json:"cachedResources"`
	CachedResourceTemplates *string `gorm:"column:cached_resource_templates;type:text" json:"cachedResourceTemplates"`
	CachedPrompts           *string `gorm:"column:cached_prompts;type:text" json:"cachedPrompts"`
	AnonymousAccess         bool    `gorm:"column:anonymous_access;not null;default:false" json:"anonymousAccess"`
	AnonymousRateLimit      int     `gorm:"column:anonymous_rate_limit;not null;default:10" json:"anonymousRateLimit"`
}

func (Server) TableName() string { return "server" }

type Log struct {
	ID                     int     `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	CreatedAt              int     `gorm:"column:created_at;not null;default:0;index" json:"createdAt"`
	Action                 int     `gorm:"column:action;not null;default:0" json:"action"`
	UserID                 string  `gorm:"column:userid;type:varchar;not null;default:'';index" json:"userid"`
	ServerID               *string `gorm:"column:server_id;type:varchar(128);index" json:"serverId"`
	SessionID              string  `gorm:"column:session_id;type:varchar;not null;default:'';index" json:"sessionId"`
	UpstreamRequestID      string  `gorm:"column:upstream_request_id;type:varchar;not null;default:''" json:"upstreamRequestId"`
	UniformRequestID       *string `gorm:"column:uniform_request_id;type:varchar;index" json:"uniformRequestId"`
	ParentUniformRequestID *string `gorm:"column:parent_uniform_request_id;type:varchar" json:"parentUniformRequestId"`
	ProxyRequestID         *string `gorm:"column:proxy_request_id;type:varchar" json:"proxyRequestId"`
	IP                     string  `gorm:"column:ip;type:varchar;not null;default:''" json:"ip"`
	UA                     string  `gorm:"column:ua;type:varchar;not null;default:''" json:"ua"`
	TokenMask              string  `gorm:"column:token_mask;type:varchar;not null;default:''" json:"tokenMask"`
	RequestParams          string  `gorm:"column:request_params;type:text;not null;default:''" json:"requestParams"`
	ResponseResult         string  `gorm:"column:response_result;type:text;not null;default:''" json:"responseResult"`
	Error                  string  `gorm:"column:error;type:text;not null;default:''" json:"error"`
	Duration               *int    `gorm:"column:duration" json:"duration"`
	StatusCode             *int    `gorm:"column:status_code" json:"statusCode"`
}

func (Log) TableName() string { return "log" }

type Event struct {
	ID          int       `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	EventID     string    `gorm:"column:event_id;type:varchar(255);not null;uniqueIndex" json:"eventId"`
	StreamID    string    `gorm:"column:stream_id;type:varchar(255);not null;index" json:"streamId"`
	SessionID   string    `gorm:"column:session_id;type:varchar(255);not null;index" json:"sessionId"`
	MessageType string    `gorm:"column:message_type;type:varchar(50);not null" json:"messageType"`
	MessageData string    `gorm:"column:message_data;not null" json:"messageData"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;index;default:now()" json:"createdAt"`
	ExpiresAt   time.Time `gorm:"column:expires_at;not null;index" json:"expiresAt"`
}

func (Event) TableName() string { return "mcp_events" }

type DnsConf struct {
	Subdomain   string `gorm:"column:subdomain;type:varchar(128);not null;default:''" json:"subdomain"`
	Type        int    `gorm:"column:type;not null;default:0" json:"type"`
	PublicIP    string `gorm:"column:public_ip;type:varchar(128);not null;default:''" json:"publicIp"`
	ID          int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Addtime     int    `gorm:"column:addtime;not null;default:0" json:"addtime"`
	UpdateTime  int    `gorm:"column:update_time;not null;default:0" json:"updateTime"`
	TunnelID    string `gorm:"column:tunnel_id;type:varchar(256);not null;default:''" json:"tunnelId"`
	ProxyID     int    `gorm:"column:proxy_id;not null;default:0" json:"proxyId"`
	CreatedBy   int    `gorm:"column:created_by;not null;default:0" json:"createdBy"`
	Credentials string `gorm:"column:credentials;type:text;not null;default:''" json:"credentials"`
}

func (DnsConf) TableName() string { return "dns_conf" }

type IPWhitelist struct {
	ID      int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	IP      string `gorm:"column:ip;type:varchar(128);not null;default:''" json:"ip"`
	Addtime int    `gorm:"column:addtime;not null;default:0" json:"addtime"`
}

func (IPWhitelist) TableName() string { return "ip_whitelist" }

type License struct {
	ID         int    `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	LicenseStr string `gorm:"column:license_str;type:text;not null" json:"licenseStr"`
	Addtime    int    `gorm:"column:addtime;not null" json:"addtime"`
	Status     int    `gorm:"column:status;not null" json:"status"`
}

func (License) TableName() string { return "license" }

type OAuthClient struct {
	ClientID                string                   `gorm:"column:client_id;primaryKey;type:varchar(255);not null" json:"clientId"`
	ClientSecret            *string                  `gorm:"column:client_secret;type:varchar(255)" json:"clientSecret"`
	TokenEndpointAuthMethod string                   `gorm:"column:token_endpoint_auth_method;type:varchar(50);not null;default:'client_secret_basic';index:idx_oauth_client_duplicate_check,priority:2" json:"tokenEndpointAuthMethod"`
	Name                    string                   `gorm:"column:name;type:varchar(255);not null;index:idx_oauth_client_duplicate_check,priority:1" json:"name"`
	RedirectUris            datatypes.JSON           `gorm:"column:redirect_uris;type:jsonb;not null" json:"redirectUris"`
	Scopes                  datatypes.JSON           `gorm:"column:scopes;type:jsonb;not null" json:"scopes"`
	GrantTypes              datatypes.JSON           `gorm:"column:grant_types;type:jsonb;not null" json:"grantTypes"`
	ResponseTypes           datatypes.JSON           `gorm:"column:response_types;type:jsonb;not null;default:'[]'" json:"responseTypes"`
	UserID                  *string                  `gorm:"column:user_id;type:varchar(64)" json:"userId"`
	Trusted                 bool                     `gorm:"column:trusted;not null;default:false" json:"trusted"`
	CreatedAt               time.Time                `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt               time.Time                `gorm:"column:updated_at;not null;default:now();autoUpdateTime" json:"updatedAt"`
	AuthorizationCodes      []OAuthAuthorizationCode `gorm:"foreignKey:ClientID;references:ClientID" json:"authorizationCodes"`
	Tokens                  []OAuthToken             `gorm:"foreignKey:ClientID;references:ClientID" json:"tokens"`
}

func (OAuthClient) TableName() string { return "oauth_clients" }

type OAuthAuthorizationCode struct {
	Code            string         `gorm:"column:code;primaryKey;type:varchar(255);not null" json:"code"`
	ClientID        string         `gorm:"column:client_id;type:varchar(255);not null;index" json:"clientId"`
	UserID          string         `gorm:"column:user_id;type:varchar(64);not null;index" json:"userId"`
	RedirectURI     string         `gorm:"column:redirect_uri;type:text;not null" json:"redirectUri"`
	Scopes          datatypes.JSON `gorm:"column:scopes;type:jsonb;not null" json:"scopes"`
	Resource        *string        `gorm:"column:resource;type:text" json:"resource"`
	CodeChallenge   *string        `gorm:"column:code_challenge;type:varchar(255)" json:"codeChallenge"`
	ChallengeMethod *string        `gorm:"column:challenge_method;type:varchar(10)" json:"challengeMethod"`
	ExpiresAt       time.Time      `gorm:"column:expires_at;not null;index" json:"expiresAt"`
	CreatedAt       time.Time      `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	Used            bool           `gorm:"column:used;not null;default:false" json:"used"`
	Client          OAuthClient    `gorm:"foreignKey:ClientID;references:ClientID;constraint:OnDelete:CASCADE" json:"client"`
}

func (OAuthAuthorizationCode) TableName() string { return "oauth_authorization_codes" }

type OAuthToken struct {
	TokenID               string         `gorm:"column:token_id;primaryKey;type:text;not null" json:"tokenId"`
	AccessToken           string         `gorm:"column:access_token;type:text;not null;uniqueIndex;index" json:"accessToken"`
	RefreshToken          *string        `gorm:"column:refresh_token;type:text;uniqueIndex;index" json:"refreshToken"`
	ClientID              string         `gorm:"column:client_id;type:varchar(255);not null;index" json:"clientId"`
	UserID                string         `gorm:"column:user_id;type:varchar(64);not null;index" json:"userId"`
	Scopes                datatypes.JSON `gorm:"column:scopes;type:jsonb;not null" json:"scopes"`
	Resource              *string        `gorm:"column:resource;type:text" json:"resource"`
	AccessTokenExpiresAt  time.Time      `gorm:"column:access_token_expires_at;not null;index" json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt *time.Time     `gorm:"column:refresh_token_expires_at" json:"refreshTokenExpiresAt"`
	CreatedAt             time.Time      `gorm:"column:created_at;not null;default:now()" json:"createdAt"`
	UpdatedAt             time.Time      `gorm:"column:updated_at;not null;default:now();autoUpdateTime" json:"updatedAt"`
	Revoked               bool           `gorm:"column:revoked;not null;default:false" json:"revoked"`
	Client                OAuthClient    `gorm:"foreignKey:ClientID;references:ClientID;constraint:OnDelete:CASCADE" json:"client"`
}

func (OAuthToken) TableName() string { return "oauth_tokens" }

type ApprovalRequest struct {
	ID               string         `gorm:"primaryKey;column:id" json:"id"`
	UserID           string         `gorm:"column:user_id;type:varchar(64);index:idx_approval_user_status,priority:1;index:idx_approval_user_created,priority:1" json:"userId"`
	ServerID         *string        `gorm:"column:server_id;type:varchar(128)" json:"serverId"`
	ToolName         string         `gorm:"column:tool_name" json:"toolName"`
	CanonicalArgs    datatypes.JSON `gorm:"column:canonical_args;type:jsonb" json:"canonicalArgs"`
	RedactedArgs     datatypes.JSON `gorm:"column:redacted_args;type:jsonb" json:"redactedArgs"`
	PolicyVersion    int            `gorm:"column:policy_version" json:"policyVersion"`
	RequestHash      string         `gorm:"column:request_hash;type:varchar(64);index:idx_approval_request_hash_status,priority:1" json:"requestHash"`
	Status           string         `gorm:"column:status;type:varchar(20);index:idx_approval_request_hash_status,priority:2;index:idx_approval_user_status,priority:2;index:idx_approval_expires_status,priority:2" json:"status"`
	ExpiresAt        time.Time      `gorm:"column:expires_at;index:idx_approval_expires_status,priority:1" json:"expiresAt"`
	DecidedAt        *time.Time     `gorm:"column:decided_at" json:"decidedAt"`
	DecisionReason   *string        `gorm:"column:decision_reason;type:text" json:"decisionReason"`
	DecidedByUserID  *string        `gorm:"column:decided_by_user_id;type:varchar(64)" json:"decidedByUserId"`
	DecidedByRole    *int           `gorm:"column:decided_by_role" json:"decidedByRole"`
	DecisionChannel  *string        `gorm:"column:decision_channel;type:varchar(32)" json:"decisionChannel"`
	ExecutedAt       *time.Time     `gorm:"column:executed_at" json:"executedAt"`
	ExecutionError   *string        `gorm:"column:execution_error;type:text" json:"executionError"`
	ExecutionResult  datatypes.JSON `gorm:"column:execution_result;type:jsonb" json:"executionResult"`
	UniformRequestID *string        `gorm:"column:uniform_request_id;type:varchar" json:"uniformRequestId"`
	CreatedAt        time.Time      `gorm:"column:created_at;autoCreateTime;index:idx_approval_user_created,priority:2" json:"createdAt"`
	UpdatedAt        time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (ApprovalRequest) TableName() string { return "approval_request" }

type ToolPolicySet struct {
	ID        string         `gorm:"primaryKey;column:id" json:"id"`
	ServerID  *string        `gorm:"column:server_id;type:varchar(128);index:idx_policy_server_status,priority:1" json:"serverId"`
	Version   int            `gorm:"column:version;default:1" json:"version"`
	Status    string         `gorm:"column:status;type:varchar(20);default:active;index:idx_policy_server_status,priority:2" json:"status"`
	Dsl       datatypes.JSON `gorm:"column:dsl;type:jsonb" json:"dsl"`
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (ToolPolicySet) TableName() string { return "tool_policy_set" }

func (t *OAuthToken) BeforeCreate(_ *gorm.DB) error {
	if t.TokenID == "" {
		t.TokenID = uuid.NewString()
	}
	return nil
}
