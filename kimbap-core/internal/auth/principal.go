package auth

import "time"

type PrincipalType string

const (
	PrincipalTypeService  PrincipalType = "service"
	PrincipalTypeOperator PrincipalType = "operator"
	PrincipalTypeSystem   PrincipalType = "system"
)

type Principal struct {
	ID        string
	Type      PrincipalType
	TenantID  string
	AgentName string
	Scopes    []string
	TokenID   string
	IssuedAt  time.Time
	ExpiresAt time.Time
}
