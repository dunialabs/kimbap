package mcp

import (
	"net/http"

	"github.com/dunialabs/kimbap-core/internal/mcp/controller"
)

type Middleware func(http.Handler) http.Handler

type MCPRouter struct {
	controller *controller.MCPController
}

func NewMCPRouter() *MCPRouter {
	return &MCPRouter{controller: controller.NewMCPController()}
}

func (r *MCPRouter) Handler(middlewares ...Middleware) http.Handler {
	var handler http.Handler = http.HandlerFunc(r.controller.HandleMCP)
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func (r *MCPRouter) RegisterRoutes(mux *http.ServeMux, middlewares ...Middleware) {
	handler := r.Handler(middlewares...)
	mux.Handle("/mcp", handler)
	mux.Handle("/mcp/", handler)
}
