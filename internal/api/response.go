package api

import (
	"encoding/json"
	"net/http"

	"github.com/dunialabs/kimbap/internal/actions"
)

// Envelope is the canonical API response envelope used by all kimbap
// REST endpoints. Both kimbap and kimbap-console converge to this shape.
//
//	Success: {"success": true,  "data": T,                                "request_id": "..."}
//	Error:   {"success": false, "error": {"code","message","retryable"},  "request_id": "..."}
type Envelope struct {
	Success   bool       `json:"success"`
	Data      any        `json:"data,omitempty"`
	Error     *ErrorBody `json:"error,omitempty"`
	RequestID string     `json:"request_id,omitempty"`
}

// ErrorBody is the structured error payload inside an Envelope.
type ErrorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

// writeSuccess writes a canonical success response with the unified envelope.
func writeSuccess(w http.ResponseWriter, r *http.Request, status int, data any) {
	env := Envelope{
		Success:   true,
		Data:      data,
		RequestID: requestIDFromContext(r.Context()),
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

// writeEnvelopeError writes a canonical error response with the unified envelope.
func writeEnvelopeError(w http.ResponseWriter, r *http.Request, execErr *actions.ExecutionError) {
	if execErr == nil {
		execErr = actions.NewExecutionError(
			actions.ErrDownstreamUnavailable, "unknown error",
			http.StatusInternalServerError, false, nil,
		)
	}
	status := execErr.HTTPStatus
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	msg := execErr.Message
	var details map[string]any
	if status >= 500 {
		msg = "internal server error"
		details = nil
	} else {
		details = execErr.Details
	}
	env := Envelope{
		Success: false,
		Error: &ErrorBody{
			Code:      execErr.Code,
			Message:   msg,
			Retryable: execErr.Retryable,
			Details:   details,
		},
		RequestID: requestIDFromContext(r.Context()),
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}
