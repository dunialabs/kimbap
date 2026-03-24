package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dunialabs/kimbap-core/internal/approvals"
	"github.com/dunialabs/kimbap-core/internal/auth"
	"github.com/dunialabs/kimbap-core/internal/runtime"
	"github.com/dunialabs/kimbap-core/internal/store"
	"github.com/dunialabs/kimbap-core/internal/webhooks"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	router            chi.Router
	store             store.Store
	addr              string
	httpServer        *http.Server
	tokenService      *auth.TokenService
	runtime           *runtime.Runtime
	approvalManager   *approvals.ApprovalManager
	webhookDispatcher *webhooks.Dispatcher
	skipConsole       bool
}

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultIdleTimeout       = 60 * time.Second
)

type ServerOption func(*Server)

func WithRuntime(rt *runtime.Runtime) ServerOption {
	return func(s *Server) {
		s.runtime = rt
	}
}

func WithoutConsole() ServerOption {
	return func(s *Server) {
		s.skipConsole = true
	}
}

func WithWebhookDispatcher(d *webhooks.Dispatcher) ServerOption {
	return func(s *Server) {
		s.webhookDispatcher = d
	}
}

func NewServer(addr string, st store.Store, opts ...ServerOption) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(RequestID())
	r.Use(JSONContentType())

	s := &Server{
		router: r,
		store:  st,
		addr:   addr,
	}
	if s.addr == "" {
		s.addr = ":8080"
	}
	if st != nil {
		s.tokenService = auth.NewTokenService(&storeTokenAdapter{st: st})
		s.approvalManager = approvals.NewApprovalManager(&storeApprovalAdapter{st: st}, nil, 10*time.Minute)
	}
	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}
	s.registerRoutes()
	return s
}

func (s *Server) Router() chi.Router {
	return s.router
}

func (s *Server) Start(ctx context.Context) error {
	if s.httpServer != nil {
		return errors.New("server already started")
	}
	s.httpServer = newHTTPServer(s.addr, s.router)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
		s.httpServer = nil
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			s.httpServer = nil
			return nil
		}
		s.httpServer = nil
		return err
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	err := s.httpServer.Shutdown(ctx)
	s.httpServer = nil
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
