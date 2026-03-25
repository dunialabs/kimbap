package socket

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	"github.com/dunialabs/kimbap-core/internal/logger"
	mcpservices "github.com/dunialabs/kimbap-core/internal/mcp/services"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/dunialabs/kimbap-core/internal/security"
	coretypes "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/dunialabs/kimbap-core/internal/user"
	socketio "github.com/googollee/go-socket.io"
	"github.com/googollee/go-socket.io/engineio"
	"github.com/rs/zerolog"
)

type tokenValidator interface {
	ValidateToken(token string) (string, error)
}

type pendingRequest struct {
	resolver  func(SocketResponse[map[string]any])
	timer     *time.Timer
	userID    string
	action    SocketActionType
	createdAt time.Time
}

type SocketService struct {
	mu              sync.RWMutex
	io              *socketio.Server
	validator       tokenValidator
	userRepo        *repository.UserRepository
	connections     map[string][]UserConnection
	connectionUsers map[string]string
	socketData      map[string]SocketData
	pendingRequests map[string]pendingRequest
	serverName      string
	serverID        string
	log             zerolog.Logger
}

func NewSocketService(validator tokenValidator, userRepo *repository.UserRepository) *SocketService {
	if userRepo == nil {
		userRepo = repository.NewUserRepository(nil)
	}
	lg := logger.CreateLogger("SocketService")
	return &SocketService{
		validator:       validator,
		userRepo:        userRepo,
		connections:     make(map[string][]UserConnection),
		connectionUsers: make(map[string]string),
		socketData:      make(map[string]SocketData),
		pendingRequests: make(map[string]pendingRequest),
		serverName:      "Kimbap Core",
		serverID:        "kimbap-core",
		log:             lg,
	}
}

func (s *SocketService) Initialize(httpServer *http.Server) error {
	if httpServer == nil {
		return errors.New("http server is required")
	}

	ioServer := socketio.NewServer(&engineio.Options{
		PingInterval: 25 * time.Second,
		PingTimeout:  60 * time.Second,
		RequestChecker: func(r *http.Request) (http.Header, error) {
			h := http.Header{}
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				if !isAllowedSocketOrigin(r, origin) {
					return nil, errors.New("forbidden origin")
				}
				h.Set("Access-Control-Allow-Origin", origin)
				h.Set("Vary", "Origin")
			}
			h.Set("Access-Control-Allow-Methods", "GET, POST")
			return h, nil
		},
	})

	if err := s.registerHandlers(ioServer); err != nil {
		return err
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error().Interface("panic", r).Msg("recovered panic in socket.io server")
			}
		}()
		if err := ioServer.Serve(); err != nil {
			s.log.Error().Err(err).Msg("socket.io server stopped")
		}
	}()

	baseHandler := httpServer.Handler
	if baseHandler == nil {
		baseHandler = http.DefaultServeMux
	}
	mux := http.NewServeMux()
	mux.Handle("/socket.io/", ioServer)
	mux.Handle("/", baseHandler)
	httpServer.Handler = mux

	s.mu.Lock()
	s.io = ioServer
	s.mu.Unlock()

	s.updateServerInfo()
	s.log.Info().Msg("socket.io server initialized")
	return nil
}

func (s *SocketService) registerHandlers(ioServer *socketio.Server) error {
	requestHandler := user.NewRequestHandler(database.DB)

	ioServer.OnConnect("/", func(conn socketio.Conn) error { return s.handleSocketConnect(conn) })

	ioServer.OnEvent("/", SocketEventClientInfo, func(conn socketio.Conn, info ClientInfo) {
		connID := conn.ID()
		s.mu.Lock()
		userID := s.connectionUsers[connID]
		for i := range s.connections[userID] {
			if s.connections[userID][i].SocketID == connID {
				s.connections[userID][i].DeviceName = info.DeviceName
				s.connections[userID][i].DeviceType = info.DeviceType
				s.connections[userID][i].AppVersion = info.AppVersion
			}
		}
		sd := s.socketData[connID]
		sd.DeviceName = info.DeviceName
		sd.DeviceType = info.DeviceType
		sd.AppVersion = info.AppVersion
		s.socketData[connID] = sd
		s.mu.Unlock()
	})

	ioServer.OnEvent("/", SocketEventClientMessage, func(conn socketio.Conn, _ map[string]any) {
		conn.Emit(SocketEventAck, map[string]any{
			"message":   "Message received",
			"timestamp": time.Now().UnixMilli(),
		})
	})

	ioServer.OnEvent("/", "get_capabilities", func(conn socketio.Conn, req map[string]any) {
		requestID, _ := req["requestId"].(string)
		userID := s.userIDByConnection(conn.ID())
		if userID == "" {
			s.emitSocketError(conn, requestID, "Unauthorized")
			return
		}

		caps, err := requestHandler.GetCapabilities(userID)
		if err != nil {
			s.emitSocketError(conn, requestID, err.Error())
			return
		}

		payload := map[string]any{"capabilities": caps}
		s.emitSocketSuccess(conn, requestID, &payload)
	})

	ioServer.OnEvent("/", "set_capabilities", func(conn socketio.Conn, req map[string]any) {
		requestID, _ := req["requestId"].(string)
		data, _ := req["data"].(map[string]any)
		if data == nil {
			data = map[string]any{}
		}
		userID := s.userIDByConnection(conn.ID())
		if userID == "" {
			s.emitSocketError(conn, requestID, "Unauthorized")
			return
		}

		if err := requestHandler.SetCapabilities(userID, data); err != nil {
			s.emitSocketError(conn, requestID, err.Error())
			return
		}

		s.emitSocketSuccess(conn, requestID, nil)
	})

	ioServer.OnEvent("/", "configure_server", func(conn socketio.Conn, req map[string]any) {
		requestID, _ := req["requestId"].(string)
		data, _ := req["data"].(map[string]any)
		if data == nil {
			data = map[string]any{}
		}
		userID := s.userIDByConnection(conn.ID())
		if userID == "" {
			s.emitSocketError(conn, requestID, "Unauthorized")
			return
		}

		s.mu.RLock()
		sd := s.socketData[conn.ID()]
		s.mu.RUnlock()
		result, err := requestHandler.ConfigureServer(userID, sd.UserToken, data)
		if err != nil {
			s.emitSocketError(conn, requestID, err.Error())
			return
		}

		s.emitSocketSuccess(conn, requestID, &result)
	})

	ioServer.OnEvent("/", "unconfigure_server", func(conn socketio.Conn, req map[string]any) {
		requestID, _ := req["requestId"].(string)
		data, _ := req["data"].(map[string]any)
		if data == nil {
			data = map[string]any{}
		}
		userID := s.userIDByConnection(conn.ID())
		if userID == "" {
			s.emitSocketError(conn, requestID, "Unauthorized")
			return
		}

		result, err := requestHandler.UnconfigureServer(userID, data)
		if err != nil {
			s.emitSocketError(conn, requestID, err.Error())
			return
		}

		s.emitSocketSuccess(conn, requestID, &result)
	})

	ioServer.OnEvent("/", "approval_decide", func(conn socketio.Conn, req map[string]any) {
		requestID, _ := req["requestId"].(string)
		data, _ := req["data"].(map[string]any)
		if data == nil {
			data = map[string]any{}
		}

		userID := s.userIDByConnection(conn.ID())
		if userID == "" {
			s.emitSocketError(conn, requestID, "Unauthorized")
			return
		}

		idRaw, _ := data["id"].(string)
		decisionRaw, _ := data["decision"].(string)
		id := strings.TrimSpace(idRaw)
		decision := strings.ToUpper(strings.TrimSpace(decisionRaw))
		if id == "" {
			s.emitSocketError(conn, requestID, "Missing required field: id")
			return
		}
		if decision != coretypes.ApprovalStatusApproved && decision != coretypes.ApprovalStatusRejected {
			s.emitSocketError(conn, requestID, "Invalid decision: must be APPROVED or REJECTED")
			return
		}

		approval, err := mcpservices.ApprovalServiceInstance().GetByID(id)
		if err != nil {
			s.emitSocketError(conn, requestID, err.Error())
			return
		}
		if approval == nil || approval.UserID != userID {
			s.emitSocketError(conn, requestID, "Approval request not found")
			return
		}

		var reason *string
		reasonRaw, _ := data["reason"].(string)
		if trimmedReason := strings.TrimSpace(reasonRaw); trimmedReason != "" {
			reason = &trimmedReason
		}

		s.mu.RLock()
		sd, sdExists := s.socketData[conn.ID()]
		s.mu.RUnlock()
		if !sdExists {
			s.emitSocketError(conn, requestID, "Session state not found")
			return
		}
		actorRole := sd.UserRole
		actor := mcpservices.ApprovalDecisionActor{
			ActorUserID: &userID,
			ActorRole:   &actorRole,
			Channel:     "socket",
		}

		result, err := mcpservices.ApprovalServiceInstance().Decide(id, decision, actor, reason)
		if err != nil {
			s.emitSocketError(conn, requestID, err.Error())
			return
		}
		if result == nil {
			s.emitSocketError(conn, requestID, "Decision failed: request not found, not PENDING, or expired")
			return
		}

		GetSocketNotifier().NotifyApprovalDecided(result.UserID, result.ID, result.ToolName, result.Status, result.DecisionReason)
		payload := map[string]any{"id": result.ID, "status": result.Status}
		s.emitSocketSuccess(conn, requestID, &payload)
	})

	ioServer.OnEvent("/", SocketEventSocketResponse, func(conn socketio.Conn, response SocketResponse[map[string]any]) {
		s.handleClientResponse(conn, response)
	})

	ioServer.OnDisconnect("/", func(conn socketio.Conn, reason string) { s.handleSocketDisconnect(conn, reason) })

	ioServer.OnError("/", func(conn socketio.Conn, err error) {
		if conn != nil {
			s.log.Error().Err(err).Str("socketId", conn.ID()).Msg("socket error")
			return
		}
		s.log.Error().Err(err).Msg("socket server error")
	})

	return nil
}

func (s *SocketService) handleSocketConnect(conn socketio.Conn) error {
	token := ""
	if conn != nil {
		req := conn.RemoteHeader()
		auth := strings.TrimSpace(req.Get("Authorization"))
		parts := strings.Fields(auth)
		if len(parts) > 1 && strings.EqualFold(parts[0], "Bearer") {
			t := strings.TrimSpace(strings.Join(parts[1:], " "))
			if t != "" {
				token = t
			}
		}
	}
	if token == "" {
		return errors.New("missing authentication token")
	}
	if s.validator == nil {
		return errors.New("token validator is not configured")
	}

	userID, err := s.validator.ValidateToken(token)
	if err != nil {
		return err
	}
	if s.userRepo == nil {
		return errors.New("user repository is not configured")
	}

	userEntity, err := s.userRepo.FindByUserID(userID)
	if err != nil {
		return err
	}
	if userEntity == nil {
		return errors.New("user not found")
	}
	if userEntity.Status != coretypes.UserStatusEnabled {
		return errors.New("user is disabled")
	}
	if userEntity.ExpiresAt > 0 && time.Now().Unix() > int64(userEntity.ExpiresAt) {
		return errors.New("User authorization has expired")
	}
	if userEntity.EncryptedToken != nil && *userEntity.EncryptedToken != "" {
		if !security.VerifyTokenAgainstEncrypted(token, *userEntity.EncryptedToken) {
			return errors.New("token has been revoked or rotated")
		}
	}

	connID := conn.ID()
	conn.Join(userID)

	s.mu.Lock()
	s.connections[userID] = append(s.connections[userID], UserConnection{
		UserID:      userID,
		SocketID:    connID,
		ConnectedAt: time.Now(),
	})
	s.connectionUsers[connID] = userID
	s.socketData[connID] = SocketData{UserID: userID, UserToken: token, UserRole: userEntity.Role}
	count := len(s.connections[userID])
	s.mu.Unlock()

	s.log.Info().Str("userId", userID).Str("socketId", connID).Int("deviceCount", count).Msg("socket connected")
	conn.Emit(SocketEventServerInfo, map[string]any{
		"serverId":   s.serverID,
		"serverName": s.serverName,
		"version":    config.AppInfo.Version,
	})

	go func(uid string) {
		defer func() {
			if r := recover(); r != nil {
				s.log.Error().Interface("panic", r).Msg("recovered panic in socket notification goroutine")
			}
		}()
		notifier := GetSocketNotifier()
		notifier.NotifyUserPermissionChanged(uid)
		notifier.NotifyOnlineSessions(uid)
	}(userID)

	return nil
}

func (s *SocketService) handleSocketDisconnect(conn socketio.Conn, reason string) {
	connID := conn.ID()
	s.mu.Lock()
	userID := s.connectionUsers[connID]
	delete(s.connectionUsers, connID)
	delete(s.socketData, connID)
	if userID != "" {
		curr := s.connections[userID]
		next := make([]UserConnection, 0, len(curr))
		for _, c := range curr {
			if c.SocketID != connID {
				next = append(next, c)
			}
		}
		if len(next) == 0 {
			delete(s.connections, userID)
		} else {
			s.connections[userID] = next
		}
	}
	s.mu.Unlock()

	if userID != "" && !s.IsUserOnline(userID) {
		s.clearUserPendingRequests(userID)
	}
	s.log.Info().Str("userId", userID).Str("socketId", connID).Str("reason", reason).Msg("socket disconnected")
}

func (s *SocketService) updateServerInfo() {
	repo := repository.NewProxyRepository(nil)
	proxy, err := repo.FindFirst()
	if err != nil || proxy == nil {
		return
	}
	s.mu.Lock()
	s.serverName = proxy.Name
	s.serverID = proxy.ProxyKey
	s.mu.Unlock()
}

func isAllowedSocketOrigin(r *http.Request, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" {
		return false
	}
	originHost := parsed.Host
	if host, _, err := net.SplitHostPort(originHost); err == nil {
		originHost = host
	}
	originHost = strings.Trim(strings.TrimSpace(originHost), "[]")

	requestHost := strings.Trim(strings.TrimSpace(r.Host), "[]")
	if host, _, err := net.SplitHostPort(requestHost); err == nil {
		requestHost = host
	}

	return strings.EqualFold(originHost, requestHost)
}

func (s *SocketService) getIO() *socketio.Server {
	s.mu.RLock()
	io := s.io
	s.mu.RUnlock()
	return io
}

func (s *SocketService) EmitToUser(userID string, event string, payload any) {
	if io := s.getIO(); io != nil {
		io.BroadcastToRoom("/", userID, event, payload)
	}
}

func (s *SocketService) EmitToSocket(socketID string, event string, payload any) {
	if io := s.getIO(); io != nil {
		io.BroadcastToRoom("/", socketID, event, payload)
	}
}

func (s *SocketService) EmitToAll(event string, payload any) {
	if io := s.getIO(); io != nil {
		io.BroadcastToNamespace("/", event, payload)
	}
}

func (s *SocketService) GetUserConnections(userID string) []UserConnection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conns := s.connections[userID]
	out := make([]UserConnection, len(conns))
	copy(out, conns)
	return out
}

func (s *SocketService) GetUserDeviceCount(userID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections[userID])
}

func (s *SocketService) IsUserOnline(userID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.connections[userID]) > 0
}

func (s *SocketService) GetOnlineUserIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.connections))
	for userID := range s.connections {
		if len(s.connections[userID]) > 0 {
			ids = append(ids, userID)
		}
	}
	return ids
}

func (s *SocketService) GetTotalConnections() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	total := 0
	for _, c := range s.connections {
		total += len(c)
	}
	return total
}

func (s *SocketService) DisconnectAll() {
	s.clearAllPendingRequests()
	s.mu.RLock()
	io := s.io
	s.mu.RUnlock()
	if io != nil {
		io.ForEach("/", "", func(conn socketio.Conn) {
			if err := conn.Close(); err != nil {
				s.log.Warn().Err(err).Str("socketId", conn.ID()).Msg("failed to close socket during disconnect all")
			}
		})
	}
	s.mu.Lock()
	s.connections = map[string][]UserConnection{}
	s.connectionUsers = map[string]string{}
	s.socketData = map[string]SocketData{}
	s.mu.Unlock()
}

func (s *SocketService) Shutdown() error {
	s.clearAllPendingRequests()
	s.mu.Lock()
	io := s.io
	s.io = nil
	s.mu.Unlock()
	if io == nil {
		return nil
	}
	// Close with 3-second timeout to prevent hanging during shutdown
	done := make(chan error, 1)
	go func() {
		done <- io.Close()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(3 * time.Second):
		s.log.Warn().Msg("Socket.IO close timed out after 3s")
		return nil
	}
}

func (s *SocketService) AddPendingRequest(requestID string, request pendingRequest) {
	s.mu.Lock()
	s.pendingRequests[requestID] = request
	s.mu.Unlock()
}

func (s *SocketService) RemovePendingRequest(requestID string) {
	s.mu.Lock()
	if req, ok := s.pendingRequests[requestID]; ok {
		if req.timer != nil {
			req.timer.Stop()
		}
		delete(s.pendingRequests, requestID)
	}
	s.mu.Unlock()
}

func (s *SocketService) handleClientResponse(conn socketio.Conn, response SocketResponse[map[string]any]) {
	if conn == nil {
		s.log.Warn().Msg("received socket response with nil connection")
		return
	}
	senderUserID := s.userIDByConnection(conn.ID())
	if senderUserID == "" {
		s.log.Warn().Str("socketId", conn.ID()).Msg("received socket response from unauthenticated connection")
		return
	}

	s.mu.Lock()
	req, ok := s.pendingRequests[response.RequestID]
	if ok && req.userID != senderUserID {
		s.mu.Unlock()
		s.log.Warn().
			Str("requestId", response.RequestID).
			Str("socketId", conn.ID()).
			Str("senderUserId", senderUserID).
			Str("expectedUserId", req.userID).
			Msg("rejected socket response due to user mismatch")
		return
	}
	if ok {
		delete(s.pendingRequests, response.RequestID)
	}
	s.mu.Unlock()
	if !ok {
		s.log.Warn().Str("requestId", response.RequestID).Msg("received response for unknown request id")
		return
	}
	if req.timer != nil {
		req.timer.Stop()
	}
	req.resolver(response)
}

func (s *SocketService) clearPendingRequests(matchFn func(pendingRequest) bool, errCode SocketErrorCode, errMsg string) int {
	s.mu.Lock()
	count := 0
	now := time.Now().UnixMilli()
	for requestID, req := range s.pendingRequests {
		if matchFn(req) {
			if req.timer != nil {
				req.timer.Stop()
			}
			req.resolver(SocketResponse[map[string]any]{
				RequestID: requestID,
				Success:   false,
				Error:     &SocketResponseError{Code: errCode, Message: errMsg},
				Timestamp: now,
			})
			delete(s.pendingRequests, requestID)
			count++
		}
	}
	s.mu.Unlock()
	return count
}

func (s *SocketService) clearUserPendingRequests(userID string) {
	if count := s.clearPendingRequests(func(r pendingRequest) bool { return r.userID == userID }, SocketErrorUserOffline, "user disconnected"); count > 0 {
		s.log.Debug().Str("userId", userID).Int("clearedCount", count).Msg("cleared pending requests for disconnected user")
	}
}

func (s *SocketService) clearAllPendingRequests() {
	if count := s.clearPendingRequests(func(pendingRequest) bool { return true }, SocketErrorServiceUnavailable, "service shutting down"); count > 0 {
		s.log.Debug().Int("count", count).Msg("cleared all pending requests")
	}
}

func (s *SocketService) userIDByConnection(connectionID string) string {
	s.mu.RLock()
	userID := s.connectionUsers[connectionID]
	s.mu.RUnlock()
	return userID
}

func (s *SocketService) emitSocketSuccess(conn socketio.Conn, requestID string, data *map[string]any) {
	response := SocketResponse[map[string]any]{
		RequestID: requestID,
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}
	conn.Emit(SocketEventSocketResponse, response)
}

func (s *SocketService) emitSocketError(conn socketio.Conn, requestID string, message string) {
	response := SocketResponse[map[string]any]{
		RequestID: requestID,
		Success:   false,
		Error: &SocketResponseError{
			Code:    SocketErrorServerError,
			Message: sanitizeSocketError(message),
		},
		Timestamp: time.Now().UnixMilli(),
	}
	conn.Emit(SocketEventSocketResponse, response)
}

func sanitizeSocketError(msg string) string {
	lower := strings.ToLower(msg)
	for _, keyword := range []string{"dial tcp", "connection refused", "no such host", "tls:", "x509:", "i/o timeout", "broken pipe", "sql:", "database", "gorm", "econnrefused", "enotfound"} {
		if strings.Contains(lower, keyword) {
			return "internal server error"
		}
	}
	return msg
}
