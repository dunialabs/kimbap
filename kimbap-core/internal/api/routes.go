package api

import (
	"net/http"

	"github.com/dunialabs/kimbap-core/internal/console"
	"github.com/go-chi/chi/v5"
)

func (s *Server) registerRoutes() {
	r := s.router
	if !s.skipConsole {
		r.Handle("/console", http.StripPrefix("/console", console.Handler()))
		r.Handle("/console/*", http.StripPrefix("/console", console.Handler()))
	}

	r.Route("/v1", func(r chi.Router) {
		r.Use(JSONContentType())
		r.Get("/health", s.handleHealth)
		r.Get("/actions", s.handleListActions)
		r.Get("/actions/{service}/{action}", s.handleDescribeAction)

		r.Group(func(r chi.Router) {
			r.Use(BearerAuth(s.tokenService))
			r.Use(TenantContext())

			r.Post("/actions/{service}/{action}:execute", s.handleExecuteAction)
			r.Post("/actions/validate", s.handleValidateAction)

			r.With(RequireScope("vault:read")).Get("/vault", s.handleListVaultKeys)

			r.With(RequireScope("tokens:write")).Post("/tokens", s.handleCreateToken)
			r.With(RequireScope("tokens:read")).Get("/tokens", s.handleListTokens)
			r.With(RequireScope("tokens:read")).Get("/tokens/{id}", s.handleInspectToken)
			r.With(RequireScope("tokens:write")).Delete("/tokens/{id}", s.handleRevokeToken)

			r.With(RequireScope("policies:read")).Get("/policies", s.handleGetPolicy)
			r.With(RequireScope("policies:write")).Put("/policies", s.handleSetPolicy)
			r.With(RequireScope("policies:read")).Post("/policies:evaluate", s.handleEvalPolicy)

			r.With(RequireScope("approvals:read")).Get("/approvals", s.handleListApprovals)
			r.With(RequireScope("approvals:write")).Post("/approvals/{id}:approve", s.handleApprove)
			r.With(RequireScope("approvals:write")).Post("/approvals/{id}:deny", s.handleDeny)

			r.With(RequireScope("audit:read")).Get("/audit", s.handleQueryAudit)
			r.With(RequireScope("audit:read")).Get("/audit/export", s.handleExportAudit)

			s.registerWebhookRoutes(r)
		})
	})
}
