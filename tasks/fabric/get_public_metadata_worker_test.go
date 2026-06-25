package fabric_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eluv-io/errors-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"

	fabric "github.com/qluvio/elv-mcp/tasks/fabric"
)

// -----------------------------------------------------------------------------
// Tests — Validation
// -----------------------------------------------------------------------------

func TestGetPublicMeta_MissingContentID(t *testing.T) {
    cfg := &config.Config{}
    args := fabric.GetPublicMetaArgs{}

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    ctx := runtime.WithTenant(context.Background(), newMockTenant())

    _, _, err := fabric.GetPublicMetaWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
    if err == nil {
        t.Fatalf("expected error for missing content_id")
    }
    if !errors.IsKind(errors.K.Invalid, err) {
        t.Fatalf("expected Invalid error, got %v", err)
    }
}

func TestGetPublicMeta_NoTenant(t *testing.T) {
    cfg := &config.Config{}
    args := fabric.GetPublicMetaArgs{ContentID: "iq__NO_TENANT"}

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    _, _, err := fabric.GetPublicMetaWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
    if err == nil {
        t.Fatalf("expected error for missing tenant")
    }
    if !errors.IsKind(errors.K.Permission, err) {
        t.Fatalf("expected Permission error, got %v", err)
    }
}

// -----------------------------------------------------------------------------
// Tests — Successful Path
// -----------------------------------------------------------------------------

func TestGetPublicMeta_Success(t *testing.T) {
    // Mock backend
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/q/iq__abc/struct/meta/user/public" {
            t.Fatalf("unexpected path: %s", r.URL.Path)
        }
        json.NewEncoder(w).Encode(map[string]any{
            "name":        "Test Name",
            "description": "Test Description",
            "extra":       map[string]any{"k": "v"},
        })
    }))
    defer srv.Close()

    cfg := &config.Config{ApiUrl: srv.URL}
    args := fabric.GetPublicMetaArgs{ContentID: "iq__abc"}

    // Mock auth
    prevAuth := auth.Auth	
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

    res, payload, err := fabric.GetPublicMetaWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res == nil || res.IsError {
        t.Fatalf("unexpected MCP error: %+v", res)
    }

    out, ok := payload.(fabric.GetPublicMetaResult)
    if !ok {
        t.Fatalf("expected GetPublicMetaResult, got %T", payload)
    }

    if out.Data["name"] != "Test Name" {
        t.Fatalf("unexpected name: %v", out.Data["name"])
    }
    if out.Data["description"] != "Test Description" {
        t.Fatalf("unexpected description: %v", out.Data["description"])
    }
}

// -----------------------------------------------------------------------------
// Tests — HTTP Error Paths
// -----------------------------------------------------------------------------

func TestGetPublicMeta_HTTPErrorCodes(t *testing.T) {
    statuses := []int{
        http.StatusBadRequest,
        http.StatusUnauthorized,
        http.StatusForbidden,
        http.StatusNotFound,
        http.StatusConflict,
        http.StatusInternalServerError,
    }

	// Mock auth
    prevAuth := auth.Auth	
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())


    for _, code := range statuses {
        t.Run(http.StatusText(code), func(t *testing.T) {
            srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(code)
            }))
            defer srv.Close()

            cfg := &config.Config{ApiUrl: srv.URL}
            args := fabric.GetPublicMetaArgs{ContentID: "iq__abc"}            

            res, payload, err := fabric.GetPublicMetaWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
            if err == nil {
                t.Fatalf("expected error for HTTP %d", code)
            }
            if res == nil || !res.IsError {
                t.Fatalf("expected MCP error result")
            }
            if payload != nil {
                t.Fatalf("expected nil payload")
            }
        })
    }
}