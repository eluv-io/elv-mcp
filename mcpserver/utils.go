package mcpserver

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	elog "github.com/eluv-io/log-go"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

var log = elog.Get("/mcpserver")

// recoverMiddleware prevents panics from crashing the server and returns HTTP 500 instead.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic in HTTP handler", "panic", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs each incoming request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		hasAuthHeader := authHeader != ""
		log.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"authorization", hasAuthHeader,
		)
		next.ServeHTTP(w, r)
	})
}

// selectiveAuthMiddleware verifies OAuth bearer tokens, resolves the tenant,
// and stores the TenantFabric in the request context.
func selectiveAuthMiddleware(
	verifier mcpauth.TokenVerifier,
	resourceMetadataURL string,
	tenants *config.TenantRegistry,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			tokenStr := extractBearerToken(r)
			if tokenStr == "" {
				w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+resourceMetadataURL+`"`)
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			tokenInfo, err := verifier(r.Context(), tokenStr, r)
			if err != nil {
				log.Warn("JWT verification failed", "error", err)
				w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+resourceMetadataURL+`", error="invalid_token"`)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			userID := strings.TrimSpace(tokenInfo.UserID)
			if userID == "" {
				log.Warn("JWT missing subject/user id")
				w.Header().Set("WWW-Authenticate", `Bearer resource_metadata="`+resourceMetadataURL+`", error="invalid_token"`)
				http.Error(w, "invalid token: missing subject", http.StatusUnauthorized)
				return
			}

			tenant, ok := tenants.Lookup(userID)
			if !ok {
				log.Warn("user not in any tenant", "user_id", userID)
				http.Error(w, "forbidden: user not authorized", http.StatusForbidden)
				return
			}

			log.Info("tenant resolved", "user_id", userID, "tenant", tenant.ID)

			ctx := runtime.WithTenant(r.Context(), tenant.Fabric)

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return h[7:]
	}
	return ""
}
