package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/dunialabs/kimbap-core/internal/admin"
	"github.com/dunialabs/kimbap-core/internal/api"
	"github.com/dunialabs/kimbap-core/internal/app"
	"github.com/dunialabs/kimbap-core/internal/config"
	"github.com/dunialabs/kimbap-core/internal/database"
	internalLog "github.com/dunialabs/kimbap-core/internal/log"
	"github.com/dunialabs/kimbap-core/internal/logger"
	"github.com/dunialabs/kimbap-core/internal/mcp/core"
	mcpservices "github.com/dunialabs/kimbap-core/internal/mcp/services"
	mcptypes "github.com/dunialabs/kimbap-core/internal/mcp/types"
	"github.com/dunialabs/kimbap-core/internal/middleware"
	"github.com/dunialabs/kimbap-core/internal/oauth"
	"github.com/dunialabs/kimbap-core/internal/repository"
	"github.com/dunialabs/kimbap-core/internal/security"
	"github.com/dunialabs/kimbap-core/internal/socket"
	"github.com/dunialabs/kimbap-core/internal/store"
	"github.com/dunialabs/kimbap-core/internal/user"
	"github.com/dunialabs/kimbap-core/internal/webhooks"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

var (
	appLog         = logger.CreateLogger("App")
	isShuttingDown bool
)

type shutdownServerManager interface {
	Shutdown(ctx context.Context)
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

	var managementStore store.Store
	enableV1 := !strings.EqualFold(config.Env("ENABLE_V1_API"), "false")
	if enableV1 {
		var err error
		managementStore, err = store.OpenPostgresStore(databaseURL)
		if err != nil {
			appLog.Warn().Err(err).Msg("failed to initialize v1 management store; /api/v1 routes disabled")
			managementStore = nil
		} else {
			autoMigrate := !strings.EqualFold(config.Env("AUTO_MIGRATE"), "false")
			if autoMigrate {
				if err := managementStore.Migrate(context.Background()); err != nil {
					appLog.Warn().Err(err).Msg("failed to migrate v1 management store; /api/v1 routes disabled")
					_ = managementStore.Close()
					managementStore = nil
				}
			}
		}
	}

	tokenValidator := security.NewTokenValidator()
	oauthValidator := security.NewOAuthTokenValidator(repository.NewOAuthTokenRepository(nil), app.OauthUserRepoAdapter{})
	authMW := middleware.NewAuthMiddleware(tokenValidator, oauthValidator, app.UserRepoAdapter{}, database.DB)
	adminMW := middleware.NewAdminAuthMiddleware(authMW)
	rateLimitService := security.NewRateLimitService()

	sessions := core.SessionStoreInstance()
	eventRepo := app.NewEventRepoAdapter()
	sessions.SetEventRepository(eventRepo)
	socketService := socket.NewSocketService(tokenValidator, repository.NewUserRepository(nil))
	socketBootstrapServer := &http.Server{Handler: http.NotFoundHandler()}
	if err := socketService.Initialize(socketBootstrapServer); err != nil {
		appLog.Warn().Err(err).Msg("socket.io initialization failed")
	}
	notifier := socket.GetSocketNotifier()
	notifier.SetSocketService(socketService)
	sessions.SetNotifier(notifier)
	servers := core.ServerManagerInstance()
	servers.Configure(app.ServerRepoAdapter{}, app.ManagerUserRepoAdapter{}, app.AuthFactoryAdapter{}, notifier)
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

	autoConnect := strings.EqualFold(config.Env("AUTO_CONNECT_ENABLED_SERVERS"), "true")
	reconcileIntervalSec := app.ParseIntDefault(config.Env("SERVER_RECONCILE_INTERVAL_SEC", "30"), 30)
	if reconcileIntervalSec < 5 {
		appLog.Warn().Int("configured", reconcileIntervalSec).Int("clamped", 5).Msg("SERVER_RECONCILE_INTERVAL_SEC too low, clamping to 5s")
		reconcileIntervalSec = 5
	}
	reconcileInterval := time.Duration(reconcileIntervalSec) * time.Second
	if autoConnect {
		servers.StartReconcileLoop(reconcileInterval)
		appLog.Info().Dur("interval", reconcileInterval).Msg("server reconciliation loop started")
	}

	runClusterJobs := !strings.EqualFold(config.Env("RUN_CLUSTER_JOBS"), "false")
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
	r.Use(app.FormParserMiddleware)

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
		AllowedHeaders:     []string{"Content-Type", "Authorization", "Accept", "last-event-id"},
		ExposedHeaders:     []string{"www-authenticate", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset", "Retry-After"},
		MaxAge:             86400,
		OptionsPassthrough: false,
	}))

	oauth.RegisterRoutes(r, oauth.RouterDependencies{
		UserValidator: app.OauthUserValidatorAdapter{Validator: tokenValidator},
		AdminAuth:     adminMW,
	})

	adminController := admin.NewController()
	r.With(adminMW.Middleware).Method(http.MethodPost, "/admin", http.HandlerFunc(adminController.HandleAdminRequest))

	userController := user.NewController()
	userAuthMW := user.NewUserAuthMiddleware(app.UserTokenValidatorAdapter{AuthMW: authMW})
	r.With(userAuthMW.Authenticate).Method(http.MethodPost, "/user", http.HandlerFunc(userController.HandleUserRequest))

	if managementStore != nil {
		webhookDispatcher := webhooks.NewDispatcher()
		v1ManagementAPI := api.NewServer("", managementStore, api.WithoutConsole(), api.WithWebhookDispatcher(webhookDispatcher))
		r.Mount("/api", v1ManagementAPI.Router())
	}

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		endpoints := map[string]any{
			"health": "/health",
			"admin":  "/admin",
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
		}
		if managementStore != nil {
			endpoints["v1"] = "/api/v1"
		}
		app.WriteJSON(w, http.StatusOK, map[string]any{
			"service":   "Kimbap Core",
			"version":   config.AppInfo.Version,
			"status":    "running",
			"endpoints": endpoints,
		})
	})

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		app.WriteJSON(w, http.StatusBadRequest, map[string]any{
			"error": "Invalid endpoint. Please use /admin for admin operations.",
		})
	})

	if socketBootstrapServer.Handler != nil {
		r.Handle("/socket.io/", socketBootstrapServer.Handler)
		r.Handle("/socket.io/*", socketBootstrapServer.Handler)
	}

	mcpservices.ApprovalServiceInstance().StartExpirySweeper(notifier)

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				app.WriteJSON(w, http.StatusInternalServerError, map[string]any{
					"status": "unhealthy",
				})
			}
		}()

		app.WriteJSON(w, http.StatusOK, map[string]any{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			"uptime":    time.Since(app.StartTime).Seconds(),
			"version":   config.AppInfo.Version,
		})
	})

	r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
		select {
		case <-servers.ReconcileDone():
			app.WriteJSON(w, http.StatusOK, map[string]any{
				"status":    "ready",
				"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			})
		default:
			app.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status": "not_ready",
				"reason": "initial server reconciliation in progress",
			})
		}
	})

	port := app.ParseIntDefault(config.Env("BACKEND_PORT", "3002"), 3002)
	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}
	var httpsServer *http.Server
	httpsStarted := false

	enableHTTPS := strings.EqualFold(config.Env("ENABLE_HTTPS"), "true")
	if enableHTTPS {
		certPath := config.Env("SSL_CERT_PATH")
		keyPath := config.Env("SSL_KEY_PATH")
		httpsPort := app.ParseIntDefault(config.Env("BACKEND_HTTPS_PORT", strconv.Itoa(port)), port)

		switch {
		case certPath == "" || keyPath == "":
			appLog.Warn().Str("certPath", certPath).Str("keyPath", keyPath).Msg("HTTPS enabled but certificate or key path is missing, falling back to HTTP")
		case !app.FileExists(certPath):
			appLog.Warn().Str("certPath", certPath).Msg("SSL cert file not found, falling back to HTTP")
		case !app.FileExists(keyPath):
			appLog.Warn().Str("keyPath", keyPath).Msg("SSL key file not found, falling back to HTTP")
		default:
			httpsServer = &http.Server{
				Addr:              ":" + strconv.Itoa(httpsPort),
				Handler:           r,
				ReadHeaderTimeout: 10 * time.Second,
				ReadTimeout:       60 * time.Second,
				WriteTimeout:      0,
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
			shutdown(sig.String(), httpServer, httpsServer, socketService, eventCleanup, sessions, servers, rateLimitService, logSyncSvc, managementStore)
		}
	}()

	select {}
}

func shutdown(
	sig string,
	httpServer *http.Server,
	httpsServer *http.Server,
	socketSvc *socket.SocketService,
	eventCleanup *core.EventCleanupService,
	sessions *core.SessionStore,
	servers shutdownServerManager,
	rateLimitService *security.RateLimitService,
	logSyncSvc *internalLog.LogSyncService,
	managementStore store.Store,
) {
	if isShuttingDown {
		return
	}
	isShuttingDown = true
	forceExit := time.AfterFunc(10*time.Second, func() { os.Exit(1) })
	defer forceExit.Stop()

	mcpservices.ApprovalServiceInstance().StopExpirySweeper()

	httpCtx, httpCancel := context.WithTimeout(context.Background(), 5*time.Second)
	_ = httpServer.Shutdown(httpCtx)
	if httpsServer != nil {
		_ = httpsServer.Shutdown(httpCtx)
	}
	httpCancel()

	if socketSvc != nil {
		socketSvc.DisconnectAll()
		socketShutdownDone := make(chan struct{})
		go func() {
			_ = socketSvc.Shutdown()
			close(socketShutdownDone)
		}()
		select {
		case <-socketShutdownDone:
		case <-time.After(2 * time.Second):
		}
	}

	eventCleanup.Stop()
	sessions.RemoveAllSessions(mcptypes.DisconnectReasonServerShutdown)
	sessions.Stop()

	serverCtx, serverCancel := context.WithTimeout(context.Background(), 3*time.Second)
	servers.Shutdown(serverCtx)
	serverCancel()

	rateLimitService.Close()
	logCtx, logCancel := context.WithTimeout(context.Background(), 2*time.Second)
	_ = internalLog.GetLogService().Shutdown(logCtx)
	logCancel()
	_ = logSyncSvc.Shutdown()
	if managementStore != nil {
		_ = managementStore.Close()
	}
	_ = database.Close()

	exitCode := 0
	if sig == "UNCAUGHT_EXCEPTION" || sig == "UNHANDLED_REJECTION" {
		exitCode = 1
	}
	os.Exit(exitCode)
}
