package api

import (
	"context"
	"errors"
	"net/http"
	"sync"
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
	mu                sync.Mutex
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
	defaultWriteTimeout      = 120 * time.Second
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
	s.mu.Lock()
	if s.httpServer != nil {
		s.mu.Unlock()
		return errors.New("server already started")
	}
	srv := newHTTPServer(s.addr, s.router)
	s.httpServer = srv
	s.mu.Unlock()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownErr := srv.Shutdown(shutdownCtx)
		s.mu.Lock()
		s.httpServer = nil
		s.mu.Unlock()
		if shutdownErr != nil && !errors.Is(shutdownErr, http.ErrServerClosed) {
			return shutdownErr
		}
		return nil
	case err := <-errCh:
		s.mu.Lock()
		s.httpServer = nil
		s.mu.Unlock()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
		ReadTimeout:       defaultReadTimeout,
		WriteTimeout:      defaultWriteTimeout,
		IdleTimeout:       defaultIdleTimeout,
	}
}

func (s *Server) Stop(ctx context.Context) error {
	s.mu.Lock()
	srv := s.httpServer
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	err := srv.Shutdown(ctx)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		_ = srv.Close()
	}
	s.mu.Lock()
	s.httpServer = nil
	s.mu.Unlock()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
