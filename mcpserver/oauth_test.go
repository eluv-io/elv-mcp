package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qluvio/elv-mcp/config"
)

func TestJWKSVerifier_InvalidToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"keys":[]}`))
	}))
	defer srv.Close()

	cfg := &config.Config{OAuthIssuer: srv.URL}
	verifier := NewTokenVerifier(cfg)

	_, err := verifier(context.Background(), "invalid.jwt.token", httptest.NewRequest("GET", "/", nil))
	if err == nil {
		t.Fatalf("expected verification error")
	}
}
