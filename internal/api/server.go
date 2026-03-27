package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/dunialabs/kimbap/internal/auth"
	"github.com/dunialabs/kimbap/internal/runtime"
	"github.com/dunialabs/kimbap/internal/store"
	"github.com/dunialabs/kimbap/internal/vault"
	"github.com/dunialabs/kimbap/internal/webhooks"
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
	vaultStore        vault.Store
	webhookDispatcher *webhooks.Dispatcher
	skipConsole       bool
}

const (
	defaultReadHeaderTimeout = 5 * time.Second
	defaultReadTimeout       = 60 * time.Second
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

func WithConsole() ServerOption {
	return func(s *Server) {
		s.skipConsole = false
	}
}

func WithWebhookDispatcher(d *webhooks.Dispatcher) ServerOption {
	return func(s *Server) {
		s.webhookDispatcher = d
	}
}

func WithVaultStore(vs vault.Store) ServerOption {
	return func(s *Server) {
		s.vaultStore = vs
	}
}

func NewServer(addr string, st store.Store, opts ...ServerOption) *Server {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(RequestID())

	s := &Server{
		router:      r,
		store:       st,
		addr:        addr,
		skipConsole: true,
	}
	if s.addr == "" {
		s.addr = ":8080"
	}
	if st != nil {
		s.tokenService = auth.NewTokenService(&storeTokenAdapter{st: st})
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
		shutdownErr := s.httpServer.Shutdown(shutdownCtx)
		if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
			_ = s.httpServer.Close()
		}
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
		ReadTimeout:       defaultReadTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
}

func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	err := s.httpServer.Shutdown(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		_ = s.httpServer.Close()
	}
	s.httpServer = nil
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
