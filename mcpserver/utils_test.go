package mcpserver

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/qluvio/elv-mcp/config"
)

func TestRecoverMiddleware_PanicsHandled(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	recoverMiddleware(h).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

func TestLoggingMiddleware_PassesThrough(t *testing.T) {
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer test")

	loggingMiddleware(h).ServeHTTP(w, r)

	if !called {
		t.Fatalf("expected handler to be called")
	}
}

func TestSelectiveAuthMiddleware_RequiresAuthOnInitialize(t *testing.T) {
	verifier := func(ctx context.Context, token string, r *http.Request) (*mcpauth.TokenInfo, error) {
		return nil, fmt.Errorf("invalid token")
	}

	mw := selectiveAuthMiddleware(verifier, "https://resource.example", &config.TenantRegistry{})

	body := []byte(`{"method":"initialize"}`)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer bad-token")
	w := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
