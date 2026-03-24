package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap-core/internal/admin"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	internalLog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/mcp"
	mcpauth "github.com/dunialabs/kimbap-core/internal/mcp/auth"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcpservices "github.com/dunialabs/kimbap-core/internal/mcp/services"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/middleware"
	"github.com/dunialabs/kimbap-core/internal/oauth"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/dunialabs/kimbap-core/internal/service"
	"github.com/dunialabs/kimbap-core/internal/socket"
	coretypes "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/dunialabs/kimbap-core/internal/user"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// latestProtocolVersion is hardcoded because the Go SDK (go-sdk v1.3.0) keeps this
// value unexported and uses "2025-06-18" as latest, while "2025-11-25" is marked
// "not yet released". We intentionally use "2025-11-25" as the current protocol
// version. Update this when the Go SDK formally exports it.
const latestProtocolVersion = "2025-11-25"

var (
	appLog         = logger.CreateLogger("App")
	isShuttingDown bool
)

type userRepoAdapter struct{}

type oauthUserRepoAdapter struct{}

type oauthUserValidatorAdapter struct {
	validator interface {
		ValidateToken(token string) (string, error)
	}
}

type serverRepoAdapter struct{}

type managerUserRepoAdapter struct{}

type authFactoryAdapter struct{}

type userTokenValidatorAdapter struct {
	authMW *middleware.AuthMiddleware
}

type coreAuthStrategyAdapter struct {
	strategy mcpauth.AuthStrategy
}

type eventRepoAdapter struct {
	repo *repository.EventRepository
}

type ipWhitelistStoreAdapter struct {
	repo *repository.IPWhitelistRepository
}

func (userRepoAdapter) FindByUserID(ctx context.Context, userID string) (*middleware.User, error) {
	_ = ctx
	entity, err := repository.NewUserRepository(nil).FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}
	encToken := ""
	if entity.EncryptedToken != nil {
		encToken = *entity.EncryptedToken
	}
	return &middleware.User{
		UserID:          entity.UserID,
		Role:            entity.Role,
		Status:          entity.Status,
		ExpiresAtUnix:   int64(entity.ExpiresAt),
		RateLimit:       entity.Ratelimit,
		Permissions:     json.RawMessage(entity.Permissions),
		UserPreferences: json.RawMessage(entity.UserPreferences),
		LaunchConfigs:   json.RawMessage(entity.LaunchConfigs),
		EncryptedToken:  encToken,
	}, nil
}

func (oauthUserRepoAdapter) FindByUserID(ctx context.Context, userID string) (*security.UserRecord, error) {
	_ = ctx
	entity, err := repository.NewUserRepository(nil).FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}
	return &security.UserRecord{
		UserID:    entity.UserID,
		Status:    entity.Status,
		ExpiresAt: int64(entity.ExpiresAt),
	}, nil
}

func (a oauthUserValidatorAdapter) ValidateUserToken(token string) (string, error) {
	if a.validator == nil {
		return "", errors.New("token validator is not configured")
	}

	userID, err := a.validator.ValidateToken(token)
	if err != nil {
		return "", err
	}

	entity, err := repository.NewUserRepository(nil).FindByUserID(userID)
	if err != nil {
		return "", err
	}
	if entity == nil {
		return "", errors.New("user not found")
	}
	if entity.Status != coretypes.UserStatusEnabled {
		return "", errors.New("user is disabled")
	}
	if entity.ExpiresAt > 0 && time.Now().Unix() > int64(entity.ExpiresAt) {
		return "", errors.New("user authorization has expired")
	}

	return userID, nil
}

func (serverRepoAdapter) FindAllEnabled(ctx context.Context) ([]database.Server, error) {
	_ = ctx
	return repository.NewServerRepository(nil).FindAllEnabled()
}

func (serverRepoAdapter) FindByServerID(ctx context.Context, serverID string) (*database.Server, error) {
	_ = ctx
	return repository.NewServerRepository(nil).FindByServerID(serverID)
}

func (serverRepoAdapter) UpdateCapabilities(ctx context.Context, serverID string, caps string) error {
	_ = ctx
	_, err := repository.NewServerRepository(nil).UpdateCapabilities(serverID, caps)
	return err
}

func (serverRepoAdapter) UpdateCapabilitiesCache(ctx context.Context, serverID string, data map[string]any) error {
	_ = ctx
	input := repository.ServerCapabilitiesCacheInput{
		Tools:             data["tools"],
		Resources:         data["resources"],
		ResourceTemplates: data["resourceTemplates"],
		Prompts:           data["prompts"],
	}
	return repository.NewServerRepository(nil).UpdateCapabilitiesCache(serverID, input)
}

func (serverRepoAdapter) UpdateTransportType(ctx context.Context, serverID string, transportType string) error {
	_ = ctx
	_, err := repository.NewServerRepository(nil).Update(serverID, map[string]any{"transport_type": transportType})
	return err
}

func (serverRepoAdapter) UpdateServerName(ctx context.Context, serverID string, serverName string) error {
	_ = ctx
	_, err := repository.NewServerRepository(nil).Update(serverID, map[string]any{"server_name": serverName})
	return err
}

func (managerUserRepoAdapter) FindByUserID(ctx context.Context, userID string) (*database.User, error) {
	_ = ctx
	return repository.NewUserRepository(nil).FindByUserID(userID)
}

func (managerUserRepoAdapter) UpdateLaunchConfigs(ctx context.Context, userID string, launchConfigs string) error {
	_ = ctx
	_, err := repository.NewUserRepository(nil).Update(userID, map[string]any{"launch_configs": launchConfigs})
	return err
}

func (managerUserRepoAdapter) UpdateUserPreferences(ctx context.Context, userID string, userPreferences string) error {
	_ = ctx
	_, err := repository.NewUserRepository(nil).Update(userID, map[string]any{"user_preferences": userPreferences})
	return err
}

func (a authFactoryAdapter) Build(ctx context.Context, server database.Server, launchConfig map[string]any, userToken string) (core.AuthStrategy, error) {
	_ = ctx
	if server.AuthType == coretypes.ServerAuthTypeApiKey {
		return nil, nil
	}

	oauthConfig, ok := launchConfig["oauth"].(map[string]any)
	if !ok || oauthConfig == nil {
		return nil, errors.New("missing OAuth configuration")
	}

	strategyConfig := make(map[string]interface{}, len(oauthConfig)+2)
	for key, value := range oauthConfig {
		strategyConfig[key] = value
	}
	if server.UseKimbapOauthConfig {
		strategyConfig["userToken"] = userToken
		strategyConfig["server"] = server
		strategy, err := mcpauth.NewKimbapAuthStrategy(strategyConfig)
		if err != nil {
			return nil, err
		}
		return coreAuthStrategyAdapter{strategy: strategy}, nil
	}

	strategy, err := mcpauth.Create(server.AuthType, strategyConfig)
	if err != nil {
		return nil, err
	}
	if strategy == nil {
		return nil, nil
	}

	return coreAuthStrategyAdapter{strategy: strategy}, nil
}

func (a userTokenValidatorAdapter) ValidateToken(token string) (*coretypes.AuthContext, error) {
	if a.authMW == nil {
		return nil, errors.New("auth middleware is not configured")
	}
	req, err := http.NewRequest(http.MethodPost, "/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	return a.authMW.AuthenticateRequest(req)
}

func (a coreAuthStrategyAdapter) GetInitialToken(ctx context.Context) (string, int64, error) {
	_ = ctx
	if a.strategy == nil {
		return "", 0, errors.New("auth strategy is not configured")
	}
	tokenInfo, err := a.strategy.GetInitialToken()
	if err != nil {
		return "", 0, err
	}
	return tokenInfo.AccessToken, tokenInfo.ExpiresAt, nil
}

func (a coreAuthStrategyAdapter) RefreshToken(ctx context.Context) (string, int64, error) {
	_ = ctx
	if a.strategy == nil {
		return "", 0, errors.New("auth strategy is not configured")
	}
	tokenInfo, err := a.strategy.RefreshToken()
	if err != nil {
		return "", 0, err
	}
	return tokenInfo.AccessToken, tokenInfo.ExpiresAt, nil
}

func (a coreAuthStrategyAdapter) GetCurrentOAuthConfig() map[string]interface{} {
	if a.strategy == nil {
		return nil
	}
	return a.strategy.GetCurrentOAuthConfig()
}

func (a coreAuthStrategyAdapter) MarkConfigAsPersisted() {
	if a.strategy == nil {
		return
	}
	a.strategy.MarkConfigAsPersisted()
}

func newEventRepoAdapter() *eventRepoAdapter {
	return &eventRepoAdapter{repo: repository.NewEventRepository(nil)}
}

func (a *eventRepoAdapter) Create(ctx context.Context, event *database.Event) error {
	if a == nil || a.repo == nil {
		return errors.New("event repository is not configured")
	}
	_, err := a.repo.Create(ctx, event)
	return err
}

func (a *eventRepoAdapter) FindByStreamIDAfter(ctx context.Context, streamID string, afterEventID string) ([]database.Event, error) {
	if a == nil || a.repo == nil {
		return nil, errors.New("event repository is not configured")
	}
	return a.repo.FindAfterEventID(ctx, streamID, afterEventID)
}

func (a *eventRepoAdapter) DeleteExpired(ctx context.Context) (int64, error) {
	if a == nil || a.repo == nil {
		return 0, errors.New("event repository is not configured")
	}
	return a.repo.DeleteExpired(ctx)
}

func (a *eventRepoAdapter) DeleteByStreamID(ctx context.Context, streamID string) (int64, error) {
	if a == nil || a.repo == nil {
		return 0, errors.New("event repository is not configured")
	}
	return a.repo.DeleteByStreamID(ctx, streamID)
}

func (a ipWhitelistStoreAdapter) LoadWhitelist(ctx context.Context) ([]string, error) {
	_ = ctx
	if a.repo == nil {
		return nil, errors.New("ip whitelist repository is not configured")
	}
	rows, err := a.repo.FindAll()
	if err != nil {
		return nil, err
	}
	ips := make([]string, 0, len(rows))
	for _, row := range rows {
		ips = append(ips, row.IP)
	}
	return ips, nil
}

func (a ipWhitelistStoreAdapter) AddIP(ctx context.Context, ip string) error {
	_ = ctx
	if a.repo == nil {
		return errors.New("ip whitelist repository is not configured")
	}
	exists, err := a.repo.Exists(ip)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = a.repo.Create(ip)
	return err
}

func (a ipWhitelistStoreAdapter) RemoveIP(ctx context.Context, ip string) error {
	_ = ctx
	if a.repo == nil {
		return errors.New("ip whitelist repository is not configured")
	}
	_, err := a.repo.DeleteByIP(ip)
	return err
}

func (a ipWhitelistStoreAdapter) ReplaceAll(ctx context.Context, ips []string) error {
	_ = ctx
	if a.repo == nil {
		return errors.New("ip whitelist repository is not configured")
	}
	_, err := a.repo.ReplaceAll(ips)
	return err
}

func main() {
	if err := run(); err != nil {
		appLog.Error().Err(err).Msg("failed to start application")
		os.Exit(1)
	}
}

func run() error {
	_ = config.AppInfo

	databaseURL := config.Env("DATABASE_URL")
	if err := database.Initialize(databaseURL); err != nil {
		return err
	}

	tokenValidator := security.NewTokenValidator()
	oauthValidator := security.NewOAuthTokenValidator(repository.NewOAuthTokenRepository(nil), oauthUserRepoAdapter{})
	authMW := middleware.NewAuthMiddleware(tokenValidator, oauthValidator, userRepoAdapter{}, database.DB)
	adminMW := middleware.NewAdminAuthMiddleware(authMW)
	rateLimitService := security.NewRateLimitService()
	rateLimitMW := middleware.NewRateLimitMiddleware(rateLimitService, 60)
	ipWhitelistRepo := repository.NewIPWhitelistRepository(nil)
	ipWhitelistService := security.NewIPWhitelistService(ipWhitelistStoreAdapter{repo: ipWhitelistRepo})
	if err := ipWhitelistService.LoadFromDB(); err != nil {
		return fmt.Errorf("failed to initialize IP whitelist service: %w", err)
	}
	ipWhitelistMW := middleware.NewIPWhitelistMiddleware(ipWhitelistService)

	sessions := core.SessionStoreInstance()
	eventRepo := newEventRepoAdapter()
	sessions.SetEventRepository(eventRepo)
	sessions.SetNotifier(socket.GetSocketNotifier())
	servers := core.ServerManagerInstance()
	servers.Configure(serverRepoAdapter{}, managerUserRepoAdapter{}, authFactoryAdapter{}, socket.GetSocketNotifier())
	go func() {
		defer func() {
			if r := recover(); r != nil {
				appLog.Error().Interface("panic", r).Msg("recovered panic in server preload goroutine")
			}
		}()
		ctx := context.Background()
		if repo := servers.Repository(); repo != nil {
			enabledServers, err := repo.FindAllEnabled(ctx)
			if err != nil {
				appLog.Error().Err(err).Msg("failed to preload servers")
				return
			}
			servers.PreloadServers(enabledServers)
		}
	}()
	// Server reconciliation loop: auto-connect enabled servers from DB.
	// In multi-replica LB deployments, each replica needs to independently maintain
	// connections to all enabled servers. This loop periodically queries the DB
	// and connects/disconnects servers as needed.
	autoConnect := strings.EqualFold(config.Env("AUTO_CONNECT_ENABLED_SERVERS"), "true")
	reconcileIntervalSec := parseIntDefault(config.Env("SERVER_RECONCILE_INTERVAL_SEC", "30"), 30)
	if reconcileIntervalSec < 5 {
		appLog.Warn().Int("configured", reconcileIntervalSec).Int("clamped", 5).Msg("SERVER_RECONCILE_INTERVAL_SEC too low, clamping to 5s")
		reconcileIntervalSec = 5
	}
	reconcileInterval := time.Duration(reconcileIntervalSec) * time.Second
	if autoConnect {
		servers.StartReconcileLoop(reconcileInterval)
		appLog.Info().Dur("interval", reconcileInterval).Msg("server reconciliation loop started")
	}
	// Cluster-wide singleton jobs: gate behind RUN_CLUSTER_JOBS env var.
	// In multi-replica LB deployments, only one instance should run these to avoid
	// duplicate DB cleanup and log sync work. Set RUN_CLUSTER_JOBS=false on non-primary replicas.
	runClusterJobs := !strings.EqualFold(config.Env("RUN_CLUSTER_JOBS"), "false")
	// Note: NewEventCleanupService() auto-starts the cleanup loop in its constructor.
	// When cluster jobs are disabled, we must stop it immediately. The instance is still
	// needed for per-session CleanupStream() calls, just not the periodic DB-wide cleanup.
	eventCleanup := core.NewEventCleanupService(eventRepo)
	if !runClusterJobs {
		eventCleanup.Stop()
		appLog.Info().Msg("Event cleanup periodic job disabled (RUN_CLUSTER_JOBS=false)")
	}
	_ = internalLog.GetLogService()
	logSyncSvc := internalLog.GetLogSyncService()
	if runClusterJobs {
		_ = logSyncSvc.Initialize()
	} else {
		appLog.Info().Msg("Log sync service disabled (RUN_CLUSTER_JOBS=false)")
	}

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)

	const maxBodyBytes int64 = 15 * 1024 * 1024
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.Body != nil {
				req.Body = http.MaxBytesReader(w, req.Body, maxBodyBytes)
			}
			next.ServeHTTP(w, req)
		})
	})
	r.Use(formParserMiddleware)

	// CORS: configurable via CORS_ALLOWED_ORIGINS env var.
	// In production LB deployments, set to a comma-separated list of allowed origins.
	// Default: "*" (allow all) for development.
	corsOrigins := []string{"*"}
	if envOrigins := strings.TrimSpace(config.Env("CORS_ALLOWED_ORIGINS")); envOrigins != "" {
		parsed := strings.Split(envOrigins, ",")
		filtered := make([]string, 0, len(parsed))
		for _, origin := range parsed {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				filtered = append(filtered, origin)
			}
		}
		if len(filtered) > 0 {
			corsOrigins = filtered
		}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:     corsOrigins,
		AllowedMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowedHeaders:     []string{"Content-Type", "Authorization", "Mcp-Session-Id", "mcp-session-id", "mcp-protocol-version", "Accept", "last-event-id"},
		ExposedHeaders:     []string{"mcp-session-id", "Mcp-Session-Id", "www-authenticate", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Retry-After"},
		MaxAge:             86400,
		OptionsPassthrough: false,
	}))

	r.Method(http.MethodPut, "/mcp", methodNotAllowedHandler())
	r.Method(http.MethodPut, "/mcp/", methodNotAllowedHandler())
	r.Method(http.MethodPatch, "/mcp", methodNotAllowedHandler())
	r.Method(http.MethodPatch, "/mcp/", methodNotAllowedHandler())
	r.Method(http.MethodHead, "/mcp", headMCPHandler())
	r.Method(http.MethodHead, "/mcp/", headMCPHandler())
	r.Method(http.MethodOptions, "/mcp", optionsMCPHandler())
	r.Method(http.MethodOptions, "/mcp/", optionsMCPHandler())

	oauth.RegisterRoutes(r, oauth.RouterDependencies{
		UserValidator: oauthUserValidatorAdapter{validator: tokenValidator},
		AdminAuth:     adminMW,
	})

	mcpHandler := mcp.NewMCPRouter().Handler(ipWhitelistMW.Middleware, authMW.Middleware, rateLimitMW.Middleware)
	r.Method(http.MethodGet, "/mcp", mcpHandler)
	r.Method(http.MethodGet, "/mcp/", mcpHandler)
	r.Method(http.MethodPost, "/mcp", mcpHandler)
	r.Method(http.MethodPost, "/mcp/", mcpHandler)
	r.Method(http.MethodDelete, "/mcp", mcpHandler)
	r.Method(http.MethodDelete, "/mcp/", mcpHandler)

	adminController := admin.NewController(ipWhitelistService)
	r.With(adminMW.Middleware).Method(http.MethodPost, "/admin", http.HandlerFunc(adminController.HandleAdminRequest))

	userController := user.NewController()
	userAuthMW := user.NewUserAuthMiddleware(userTokenValidatorAdapter{authMW: authMW})
	r.With(userAuthMW.Authenticate).Method(http.MethodPost, "/user", http.HandlerFunc(userController.HandleUserRequest))

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"service": "Kimbap Core",
			"version": config.AppInfo.Version,
			"status":  "running",
			"endpoints": map[string]any{
				"health":   "/health",
				"mcp":      "/mcp",
				"admin":    "/admin",
				"socketio": "/socket.io",
				"oauth": map[string]any{
					"metadata": map[string]any{
						"authorization_server": "/.well-known/oauth-authorization-server",
						"protected_resource":   "/.well-known/oauth-protected-resource",
					},
					"register":   "/register",
					"authorize":  "/authorize",
					"token":      "/token",
					"introspect": "/introspect",
					"revoke":     "/revoke",
					"admin":      "/oauth/admin/clients",
				},
			},
		})
	})

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid endpoint. Please use /mcp for MCP requests or /admin for admin operations.",
		})
	})

	var socketService *socket.SocketService
	socketBootstrapServer := &http.Server{Handler: http.NotFoundHandler()}
	socketService = socket.NewSocketService(tokenValidator, repository.NewUserRepository(nil))
	if err := socketService.Initialize(socketBootstrapServer); err != nil {
		appLog.Warn().Err(err).Msg("socket.io initialization failed")
	} else if socketBootstrapServer.Handler != nil {
		r.Handle("/socket.io/", socketBootstrapServer.Handler)
		r.Handle("/socket.io/*", socketBootstrapServer.Handler)
	}
	socket.GetSocketNotifier().SetSocketService(socketService)
	mcpservices.ApprovalServiceInstance().StartExpirySweeper(socket.GetSocketNotifier())

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"status": "unhealthy",
				})
			}
		}()

		writeJSON(w, http.StatusOK, map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			"uptime":    time.Since(startTime).Seconds(),
		})
	})

	// Readiness endpoint: used by LB to gate traffic until this replica is ready.
	// Returns 503 until the first server reconciliation completes (if enabled),
	// then returns 200. If reconciliation is not enabled, always returns 200.
	r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
		select {
		case <-servers.ReconcileDone():
			writeJSON(w, http.StatusOK, map[string]any{
				"status":    "ready",
				"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			})
		default:
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"reason": "initial server reconciliation in progress",
			})
		}
	})

	port := parseIntDefault(config.Env("BACKEND_PORT", "3002"), 3002)
	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second, // Slowloris protection
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0, // Must be 0: SSE streams are long-lived responses
		IdleTimeout:       120 * time.Second,
	}
	var httpsServer *http.Server
	httpsStarted := false

	enableHTTPS := strings.EqualFold(config.Env("ENABLE_HTTPS"), "true")
	if enableHTTPS {
		certPath := config.Env("SSL_CERT_PATH")
		keyPath := config.Env("SSL_KEY_PATH")
		httpsPort := parseIntDefault(config.Env("BACKEND_HTTPS_PORT", strconv.Itoa(port)), port)

		switch {
		case certPath == "" || keyPath == "":
			appLog.Warn().Str("certPath", certPath).Str("keyPath", keyPath).Msg("HTTPS enabled but certificate or key path is missing, falling back to HTTP")
		case !fileExists(certPath):
			appLog.Warn().Str("certPath", certPath).Msg("SSL cert file not found, falling back to HTTP")
		case !fileExists(keyPath):
			appLog.Warn().Str("keyPath", keyPath).Msg("SSL key file not found, falling back to HTTP")
		default:
			httpsServer = &http.Server{
				Addr:              ":" + strconv.Itoa(httpsPort),
				Handler:           r,
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      0, // Must be 0: SSE streams are long-lived responses
				IdleTimeout:       120 * time.Second,
			}
			go func() {
				defer func() {
					if r := recover(); r != nil {
						appLog.Error().Interface("panic", r).Msg("recovered panic in HTTPS server goroutine")
					}
				}()
				appLog.Info().Int("port", httpsPort).Str("protocol", "https").Msg("Kimbap Core HTTPS server listening")
				if err := httpsServer.ListenAndServeTLS(certPath, keyPath); err != nil && !errors.Is(err, http.ErrServerClosed) {
					appLog.Fatal().Err(err).Msg("https server failed")
				}
			}()
			httpsStarted = true
		}
	}

	if !httpsStarted {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					appLog.Error().Interface("panic", r).Msg("recovered panic in HTTP server goroutine")
				}
			}()
			appLog.Info().Int("port", port).Str("protocol", "http").Msg("Kimbap Core HTTP server listening")
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				appLog.Fatal().Err(err).Msg("http server failed")
			}
		}()
	}

	cloudflared := service.NewCloudflaredService()
	if runClusterJobs {
		cloudflared.AutoStartIfConfigExists()
	} else {
		appLog.Info().Msg("Cloudflared auto-start disabled (RUN_CLUSTER_JOBS=false)")
	}

	shutdown := func(sig string) {
		if isShuttingDown {
			return
		}
		isShuttingDown = true
		forceExit := time.AfterFunc(10*time.Second, func() { os.Exit(1) })
		defer forceExit.Stop()

		mcpservices.ApprovalServiceInstance().StopExpirySweeper()

		if socketService != nil {
			socketService.DisconnectAll()
			socketShutdownDone := make(chan struct{})
			go func() {
				_ = socketService.Shutdown()
				close(socketShutdownDone)
			}()
			select {
			case <-socketShutdownDone:
			case <-time.After(3 * time.Second):
				appLog.Warn().Msg("Socket.IO shutdown timed out after 3s")
			}
		}

		httpCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
		_ = httpServer.Shutdown(httpCtx)
		if httpsServer != nil {
			_ = httpsServer.Shutdown(httpCtx)
		}
		httpCancel()

		eventCleanup.Stop()
		sessions.RemoveAllSessions(mcptypes.DisconnectReasonServerShutdown)
		sessions.Stop()

		serverCtx, serverCancel := context.WithTimeout(context.Background(), 3*time.Second)
		servers.Shutdown(serverCtx)
		serverCancel()

		rateLimitService.Close()
		_, _ = cloudflared.StopCloudflared()

		logCtx, logCancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = internalLog.GetLogService().Shutdown(logCtx)
		logCancel()
		_ = logSyncSvc.Shutdown()
		_ = database.Close()

		exitCode := 0
		if sig == "UNCAUGHT_EXCEPTION" || sig == "UNHANDLED_REJECTION" {
			exitCode = 1
		}
		os.Exit(exitCode)
	}

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				appLog.Error().Interface("panic", r).Msg("recovered panic in signal handler goroutine")
				os.Exit(1)
			}
		}()
		for sig := range signalCh {
			shutdown(sig.String())
		}
	}()

	select {}
}

var startTime = time.Now()

func parseIntDefault(raw string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func formParserMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			_ = r.ParseForm()
		}
		next.ServeHTTP(w, r)
	})
}

func methodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32000,"message":"Method not allowed."},"id":null}`))
	}
}

func headMCPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		parts := strings.Fields(authHeader)
		hasToken := len(parts) > 1 && strings.EqualFold(parts[0], "Bearer") && strings.TrimSpace(strings.Join(parts[1:], " ")) != ""
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Expose-Headers", "Mcp-Session-Id,mcp-session-id,www-authenticate")
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("mcp-protocol-version", latestProtocolVersion)
		if !hasToken {
			w.Header().Set("WWW-Authenticate", middleware.BuildWWWAuthenticateHeader(r, "invalid_token", "Missing Authorization header"))
			writeJSON(w, http.StatusUnauthorized, map[string]any{
				"jsonrpc": "2.0",
				"error": map[string]any{
					"code":    -32000,
					"message": "Method not allowed.",
				},
				"id": nil,
			})
			return
		}
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"jsonrpc": "2.0",
			"error": map[string]any{
				"code":    -32000,
				"message": "Method not allowed.",
			},
			"id": nil,
		})
	}
}

func optionsMCPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// CORS origin is handled by the CORS middleware for normal requests.
		// For OPTIONS preflight, the CORS middleware also handles it since
		// OptionsPassthrough is false. This handler is a fallback for
		// MCP-specific preflight details.
		h.Set("Access-Control-Allow-Methods", "GET, POST, DELETE")
		reqHeaders := r.Header.Get("Access-Control-Request-Headers")
		if reqHeaders == "" {
			reqHeaders = "Content-Type, Authorization, Mcp-Session-Id, mcp-session-id, mcp-protocol-version, Accept, last-event-id"
		}
		h.Set("Access-Control-Allow-Headers", reqHeaders)
		h.Set("Access-Control-Expose-Headers", "Mcp-Session-Id,mcp-session-id,www-authenticate")
		h.Set("Access-Control-Max-Age", "86400")
		h.Set("Vary", "Access-Control-Request-Headers")
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
