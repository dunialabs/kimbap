package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/middleware"
	types "github.com/dunialabs/kimbap-core/internal/types"
)

type Controller struct {
	handler *RequestHandler
}

const maxUserRequestBodyBytes int64 = 1 << 20

func NewController() *Controller {
	return &Controller{handler: NewRequestHandler(database.DB)}
}

func (c *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/user", c.HandleUserRequest)
}

func (c *Controller) HandleUserRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action UserActionType `json:"action"`
		Data   map[string]any `json:"data"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUserRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			middleware.WriteRequestEntityTooLargeLikeExpress(w)
			return
		}
		respondInvalidJSONLikeExpress(w, err)
		return
	}
	ctxVal := r.Context().Value(middleware.AuthContextKey)
	auth, ok := ctxVal.(*types.AuthContext)
	if !ok || auth == nil {
		if value, ok2 := ctxVal.(types.AuthContext); ok2 {
			auth = &value
		}
	}
	if auth == nil {
		respondUser(w, http.StatusUnauthorized, UserResponse{Success: false, Error: &UserRespErr{Code: UserErrorUnauthorized, Message: "Unauthorized"}})
		return
	}
	token := ""
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	parts := strings.Fields(authHeader)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		token = strings.TrimSpace(parts[1])
	}
	if req.Data == nil {
		req.Data = map[string]any{}
	}

	var (
		result any
		err    error
	)

	switch req.Action {
	case GetCapabilities:
		result, err = c.handler.GetCapabilities(auth.UserID)
	case SetCapabilities:
		err = c.handler.SetCapabilities(auth.UserID, req.Data)
		result = map[string]any{"message": "Capabilities updated successfully"}
	case ConfigureServer:
		if token == "" {
			respondUser(w, http.StatusUnauthorized, UserResponse{Success: false, Error: &UserRespErr{Code: UserErrorUnauthorized, Message: "Unauthorized"}})
			return
		}
		result, err = c.handler.ConfigureServer(auth.UserID, token, req.Data)
	case UnconfigureServer:
		result, err = c.handler.UnconfigureServer(auth.UserID, req.Data)
	case GetOnlineSessions:
		result, err = c.handler.GetOnlineSessions(auth.UserID)
	default:
		respondUser(w, http.StatusBadRequest, UserResponse{Success: false, Error: &UserRespErr{Code: UserErrorInvalidRequest, Message: "Unknown action type"}})
		return
	}

	if err != nil {
		if ue, ok := err.(*UserError); ok {
			httpStatus := http.StatusBadRequest
			msg := ue.Message
			if ue.Code >= 5000 {
				httpStatus = http.StatusInternalServerError
				msg = "internal server error"
			}
			respondUser(w, httpStatus, UserResponse{Success: false, Error: &UserRespErr{Code: ue.Code, Message: msg}})
			return
		}
		respondUser(w, http.StatusInternalServerError, UserResponse{Success: false, Error: &UserRespErr{Code: UserErrorInternal, Message: "internal server error"}})
		return
	}
	respondUser(w, http.StatusOK, UserResponse{Success: true, Data: result})
}

func respondUser(w http.ResponseWriter, status int, body UserResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondInvalidJSONLikeExpress(w http.ResponseWriter, err error) {
	for _, header := range []string{
		"Access-Control-Allow-Origin",
		"Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers",
		"Access-Control-Expose-Headers",
		"Access-Control-Max-Age",
		"Vary",
	} {
		w.Header().Del(header)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_, _ = w.Write([]byte("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><title>Error</title></head><body><pre>SyntaxError: invalid JSON in request body</pre></body></html>"))
}
