package admin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/dunialabs/kimbap-core/internal/admin/handlers"
	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	"github.com/dunialabs/kimbap-core/internal/middleware"
	"github.com/dunialabs/kimbap-core/internal/service"
	types "github.com/dunialabs/kimbap-core/internal/types"
)

type Controller struct {
	userHandler     *handlers.UserHandler
	serverHandler   *handlers.ServerHandler
	queryHandler    *handlers.QueryHandler
	proxyHandler    *handlers.ProxyHandler
	logHandler      *handlers.LogHandler
	skillsHandler   *handlers.SkillsHandler
	policyHandler   *handlers.PolicyHandler
	approvalHandler *handlers.ApprovalHandler
}

const maxAdminRequestBodyBytes int64 = 1 << 20

func NewController() *Controller {
	serverManager := core.ServerManagerInstance()
	sessionStore := core.SessionStoreInstance()
	socketNotifier := serverManager.Notifier()
	if socketNotifier == nil {
		socketNotifier = core.NewNoopSocketNotifier()
	}
	return &Controller{
		userHandler:     handlers.NewUserHandler(database.DB, sessionStore, socketNotifier, serverManager),
		serverHandler:   handlers.NewServerHandler(database.DB, serverManager, sessionStore, socketNotifier),
		queryHandler:    handlers.NewQueryHandler(database.DB),
		proxyHandler:    handlers.NewProxyHandler(database.DB, sessionStore, serverManager, socketNotifier),
		logHandler:      handlers.NewLogHandler(database.DB),
		skillsHandler:   handlers.NewSkillsHandler(service.NewSkillsService(), database.DB, serverManager),
		policyHandler:   handlers.NewPolicyHandler(database.DB),
		approvalHandler: handlers.NewApprovalHandler(socketNotifier),
	}
}

func (c *Controller) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/admin", c.HandleAdminRequest)
}

func (c *Controller) HandleAdminRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAdminRequestBodyBytes)
	var request struct {
		Action int            `json:"action"`
		Data   map[string]any `json:"data"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&request); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			middleware.WriteRequestEntityTooLargeLikeExpress(w)
			return
		}
		respond(w, http.StatusBadRequest, types.AdminResponse{Success: false, Error: &types.AdminResponseError{Code: types.AdminErrorCodeInvalidRequest, Message: "Invalid admin request format"}})
		return
	}
	var extra json.RawMessage
	if dec.Decode(&extra) != io.EOF {
		respond(w, http.StatusBadRequest, types.AdminResponse{Success: false, Error: &types.AdminResponseError{Code: types.AdminErrorCodeInvalidRequest, Message: "Invalid admin request format"}})
		return
	}
	if request.Action == 0 {
		respond(w, http.StatusBadRequest, types.AdminResponse{Success: false, Error: &types.AdminResponseError{Code: types.AdminErrorCodeInvalidRequest, Message: "Invalid admin request format"}})
		return
	}
	if request.Data == nil {
		request.Data = map[string]any{}
	}

	token := ""
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	authParts := strings.Fields(authHeader)
	if len(authParts) == 2 && strings.EqualFold(authParts[0], "Bearer") && strings.TrimSpace(authParts[1]) != "" {
		token = authParts[1]
	}

	publicAction := isPublicAdminAction(request.Action)
	isLoopback := isLoopbackRequest(r)
	isLoopbackHost := isLoopbackHost(r.Host)

	var auth *types.AuthContext

	if !publicAction || !isLoopback || !isLoopbackHost {
		if token == "" {
			respondError(w, &types.AdminError{Message: "Token is required", Code: types.AdminErrorCodeForbidden})
			return
		}
		ctxVal := r.Context().Value(middleware.AuthContextKey)
		authCtx, ok := ctxVal.(*types.AuthContext)
		if !ok || authCtx == nil {
			if value, ok2 := ctxVal.(types.AuthContext); ok2 {
				authCtx = &value
			}
		}
		auth = authCtx
		if auth == nil || (auth.Role != types.UserRoleOwner && auth.Role != types.UserRoleAdmin) {
			respondError(w, &types.AdminError{Message: "Only Owner and Admin role can perform admin operations.", Code: types.AdminErrorCodeForbidden})
			return
		}

		if auth.Role == types.UserRoleOwner {
			core.ServerManagerInstance().SetOwnerToken(token)
		}

		if isOwnerOnlyAdminAction(request.Action) && auth.Role != types.UserRoleOwner {
			respondError(w, &types.AdminError{Message: "Only Owner role can perform this operation.", Code: types.AdminErrorCodeForbidden})
			return
		}
	}

	var (
		result any
		err    error
	)

	switch request.Action {
	case types.AdminActionDisableUser:
		result, err = c.userHandler.DisableUser(request.Data)
	case types.AdminActionUpdateUserPermissions:
		result, err = c.userHandler.UpdateUserPermissions(request.Data)
	case types.AdminActionCreateUser:
		result, err = c.userHandler.CreateUser(request.Data, token)
	case types.AdminActionGetUsers:
		result, err = c.userHandler.GetUsers(request.Data)
	case types.AdminActionUpdateUser:
		result, err = c.userHandler.UpdateUser(request.Data)
	case types.AdminActionDeleteUser:
		result, err = c.userHandler.DeleteUser(request.Data)
	case types.AdminActionDeleteUsersByProxy:
		result, err = c.userHandler.DeleteUsersByProxy(request.Data)
	case types.AdminActionCountUsers:
		result, err = c.userHandler.CountUsers(request.Data)
	case types.AdminActionGetOwner:
		result, err = c.userHandler.GetOwner()

	case types.AdminActionStartServer:
		result, err = c.serverHandler.StartServer(request.Data, token)
	case types.AdminActionStopServer:
		result, err = c.serverHandler.StopServer(request.Data)
	case types.AdminActionUpdateServerCapabilities:
		result, err = c.serverHandler.UpdateServerCapabilities(request.Data)
	case types.AdminActionUpdateServerLaunchCmd:
		result, err = c.serverHandler.UpdateServerLaunchCmd(request.Data, token)
	case types.AdminActionConnectAllServers:
		result, err = c.serverHandler.ConnectAllServers(token)
	case types.AdminActionCreateServer:
		result, err = c.serverHandler.CreateServer(request.Data, token)
	case types.AdminActionGetServers:
		result, err = c.serverHandler.GetServers(request.Data)
	case types.AdminActionUpdateServer:
		result, err = c.serverHandler.UpdateServer(request.Data, token)
	case types.AdminActionDeleteServer:
		result, err = c.serverHandler.DeleteServer(request.Data)
	case types.AdminActionDeleteServersByProxy:
		result, err = c.serverHandler.DeleteServersByProxy(request.Data)
	case types.AdminActionCountServers:
		result, err = c.serverHandler.CountServers()

	case types.AdminActionGetAvailableServersCapabilities:
		result, err = c.queryHandler.GetAvailableServersCapabilities()
	case types.AdminActionGetUserAvailableServersCapabilities:
		result, err = c.queryHandler.GetUserAvailableServersCapabilities(request.Data)
	case types.AdminActionGetServersStatus:
		result, err = c.queryHandler.GetServersStatus()
	case types.AdminActionGetServersCapabilities:
		result, err = c.queryHandler.GetServerCapabilities(request.Data)

	case types.AdminActionGetProxy:
		result, err = c.proxyHandler.GetProxy()
	case types.AdminActionCreateProxy:
		result, err = c.proxyHandler.CreateProxy(request.Data)
	case types.AdminActionUpdateProxy:
		result, err = c.proxyHandler.UpdateProxy(request.Data)
	case types.AdminActionDeleteProxy:
		result, err = c.proxyHandler.DeleteProxy(request.Data)
	case types.AdminActionStopProxy:
		result, err = c.proxyHandler.StopProxy()

	case types.AdminActionSetLogWebhookURL:
		result, err = c.logHandler.SetLogWebhookURL(request.Data)
	case types.AdminActionGetLogs:
		result, err = c.logHandler.GetLogs(request.Data)

	case types.AdminActionCreatePolicySet:
		result, err = c.policyHandler.CreatePolicySet(request.Data)
	case types.AdminActionGetPolicySets:
		result, err = c.policyHandler.GetPolicySets(request.Data)
	case types.AdminActionUpdatePolicySet:
		result, err = c.policyHandler.UpdatePolicySet(request.Data)
	case types.AdminActionDeletePolicySet:
		result, err = c.policyHandler.DeletePolicySet(request.Data)
	case types.AdminActionGetEffectivePolicy:
		result, err = c.policyHandler.GetEffectivePolicy(request.Data)

	case types.AdminActionListApprovalRequests:
		result, err = c.approvalHandler.ListApprovalRequests(request.Data)
	case types.AdminActionGetApprovalRequest:
		result, err = c.approvalHandler.GetApprovalRequest(request.Data)
	case types.AdminActionDecideApprovalRequest:
		result, err = c.approvalHandler.DecideApprovalRequest(request.Data, auth)
	case types.AdminActionGetPendingApprovalsCount:
		result, err = c.approvalHandler.GetPendingApprovalsCount(request.Data)

	case types.AdminActionListSkills:
		result, err = c.skillsHandler.ListSkills(request.Data)
	case types.AdminActionUploadSkill:
		result, err = c.skillsHandler.UploadSkill(request.Data, token)
	case types.AdminActionDeleteSkill:
		result, err = c.skillsHandler.DeleteSkill(request.Data, token)
	case types.AdminActionDeleteServerSkills:
		result, err = c.skillsHandler.DeleteServerSkills(request.Data, token)

	default:
		respond(w, http.StatusBadRequest, types.AdminResponse{Success: false, Error: &types.AdminResponseError{Code: types.AdminErrorCodeInvalidRequest, Message: fmt.Sprintf("Unknown action type: %d", request.Action)}})
		return
	}

	if err != nil {
		if ae, ok := err.(*types.AdminError); ok {
			respondError(w, ae)
			return
		}
		respond(w, http.StatusInternalServerError, types.AdminResponse{Success: false, Error: &types.AdminResponseError{Code: types.AdminErrorCodeInvalidRequest, Message: "internal server error"}})
		return
	}

	data := result
	if data == nil {
		data = map[string]any{}
	}
	respond(w, http.StatusOK, types.AdminResponse{Success: true, Data: data})
}

func isPublicAdminAction(action int) bool {
	_ = action
	return false
}

func isOwnerOnlyAdminAction(action int) bool {
	switch action {
	case types.AdminActionStartServer,
		types.AdminActionUpdateServerLaunchCmd,
		types.AdminActionConnectAllServers,
		types.AdminActionCreateServer,
		types.AdminActionUpdateServer,
		types.AdminActionDeleteProxy,
		types.AdminActionStopProxy,
		types.AdminActionSetLogWebhookURL,
		types.AdminActionGetLogs,
		types.AdminActionUploadSkill,
		types.AdminActionDeleteSkill,
		types.AdminActionDeleteServerSkills:
		return true
	default:
		return false
	}
}

func respond(w http.ResponseWriter, status int, body types.AdminResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func respondError(w http.ResponseWriter, err *types.AdminError) {
	status := adminHTTPStatusFromCode(err.Code)
	msg := err.Message
	if status >= 500 {
		msg = "internal server error"
	}
	respond(w, status, types.AdminResponse{Success: false, Error: &types.AdminResponseError{Code: err.Code, Message: msg}})
}

func adminHTTPStatusFromCode(code int) int {
	switch code {
	case types.AdminErrorCodeInvalidRequest:
		return http.StatusBadRequest
	case types.AdminErrorCodeForbidden:
		return http.StatusForbidden
	case types.AdminErrorCodeUserNotFound,
		types.AdminErrorCodeServerNotFound,
		types.AdminErrorCodeProxyNotFound,
		types.AdminErrorCodeSkillNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func isLoopbackRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return false
	}
	if r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "" {
		return false
	}
	return true
}

func isLoopbackHost(rawHost string) bool {
	host := strings.TrimSpace(rawHost)
	if host == "" {
		return false
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}
