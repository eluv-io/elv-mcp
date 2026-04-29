package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"

	elvauth "github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
)

// NewTokenVerifier returns an mcpauth.TokenVerifier that validates incoming
// bearer tokens as JWTs signed by the configured OAuth issuer. The issuer's
// JWKS is fetched and cached automatically.
func NewTokenVerifier(cfg *config.Config) mcpauth.TokenVerifier {
	verifier := elvauth.NewJWKSVerifier(cfg.OAuthIssuer)

	return func(ctx context.Context, tokenStr string, req *http.Request) (*mcpauth.TokenInfo, error) {
		token, err := verifier.VerifyJWT(tokenStr)
		if err != nil {
			log.Warn("JWT verification failed", "error", err)
			return nil, fmt.Errorf("unauthorized: %w", err)
		}

		info := &mcpauth.TokenInfo{
			Expiration: time.Now().Add(time.Hour),
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if exp, err := claims.GetExpirationTime(); err == nil && exp != nil {
				info.Expiration = exp.Time
			}
			if sub, err := claims.GetSubject(); err == nil {
				info.UserID = sub
			}
		}

		log.Info("authenticated request", "user_id", info.UserID)

		return info, nil
	}
}
