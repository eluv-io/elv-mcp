package mcpserver

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"

	"github.com/qluvio/elv-mcp-experiment/types"
)

// NewTokenVerifier returns an auth.TokenVerifier that logs the token and
// accepts it unconditionally. Replace with real JWT/JWKS validation later.
func NewTokenVerifier(cfg *types.Config) auth.TokenVerifier {
	return func(ctx context.Context, token string, req *http.Request) (*auth.TokenInfo, error) {
		log.Printf("[oauth] received bearer token: %s", token)
		return &auth.TokenInfo{
			Expiration: time.Now().Add(time.Hour),
		}, nil
	}
}
