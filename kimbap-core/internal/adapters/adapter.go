package adapters

import (
	"context"
	"time"

	"github.com/dunialabs/kimbap-core/internal/actions"
)

type Adapter interface {
	Type() string
	Validate(def actions.ActionDefinition) error
	Execute(ctx context.Context, req AdapterRequest) (*AdapterResult, error)
}

type AdapterRequest struct {
	Action      actions.ActionDefinition
	Input       map[string]any
	Credentials *actions.ResolvedCredentialSet
	RequestID   string
	Timeout     time.Duration
}

type AdapterResult struct {
	Output     map[string]any
	HTTPStatus int
	Headers    map[string]string
	DurationMS int64
	Retryable  bool
	RawBody    []byte
}
