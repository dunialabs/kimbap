package oauth

import (
	"encoding/json"
	"net/http"

	"github.com/dunialabs/kimbap-core/internal/middleware"
	oauthcontroller "github.com/dunialabs/kimbap-core/internal/oauth/controller"
	"github.com/dunialabs/kimbap-core/internal/oauth/service"
	types "github.com/dunialabs/kimbap-core/internal/types"
	"github.com/go-chi/chi/v5"
)

type RouterDependencies struct {
	OAuthService  *service.OAuthService
	ClientService *service.OAuthClientService
	UserValidator oauthcontroller.UserTokenValidator
	AdminAuth     *middleware.AdminAuthMiddleware
}

func RegisterRoutes(r chi.Router, deps RouterDependencies) {
	oauthSvc := deps.OAuthService
	if oauthSvc == nil {
		oauthSvc = service.NewOAuthService(nil)
	}
	clientSvc := deps.ClientService
	if clientSvc == nil {
		clientSvc = service.NewOAuthClientService(nil, oauthSvc)
	}

	oauthCtrl := oauthcontroller.NewOAuthController(oauthSvc, clientSvc, deps.UserValidator)
	metaCtrl := oauthcontroller.NewOAuthMetadataController(oauthSvc)
	clientCtrl := oauthcontroller.NewOAuthClientController(clientSvc)

	r.Get("/.well-known/oauth-authorization-server", metaCtrl.AuthorizationServerMetadata)
	r.Get("/.well-known/oauth-protected-resource", metaCtrl.ProtectedResourceMetadata)
	r.Options("/.well-known/oauth-authorization-server", metaCtrl.HandleOptions)
	r.Options("/.well-known/oauth-protected-resource", metaCtrl.HandleOptions)

	r.Post("/register", oauthCtrl.Register)
	r.Get("/authorize", oauthCtrl.ShowAuthorizePage)
	r.Post("/authorize", oauthCtrl.Authorize)
	r.Post("/token", oauthCtrl.Token)
	r.Post("/introspect", oauthCtrl.Introspect)
	r.Post("/revoke", oauthCtrl.Revoke)
	r.Options("/register", oauthCtrl.HandleOptions)
	r.Options("/authorize", oauthCtrl.HandleOptions)
	r.Options("/token", oauthCtrl.HandleOptions)
	r.Options("/introspect", oauthCtrl.HandleOptions)
	r.Options("/revoke", oauthCtrl.HandleOptions)

	r.Group(func(ar chi.Router) {
		if deps.AdminAuth != nil {
			ar.Use(deps.AdminAuth.Middleware)
		}
		ar.Use(requireAdminContext)
		ar.Get("/oauth/admin/clients", clientCtrl.ListClients)
		ar.Get("/oauth/admin/clients/{clientId}", clientCtrl.GetClient)
		ar.Put("/oauth/admin/clients/{clientId}", clientCtrl.UpdateClient)
		ar.Delete("/oauth/admin/clients/{clientId}", clientCtrl.DeleteClient)
	})
}

func requireAdminContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, _ := middleware.GetAuthContext(r.Context())
		if auth == nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("WWW-Authenticate", `Bearer realm="kimbap-core"`)
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "invalid_request",
				"error_description": "Authorization header with Bearer token is required",
			})
			return
		}
		if auth.Role != types.UserRoleOwner && auth.Role != types.UserRoleAdmin {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error":             "access_denied",
				"error_description": "Admin access required",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
