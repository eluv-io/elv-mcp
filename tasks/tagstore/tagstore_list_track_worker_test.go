package tagstore_test

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
	"github.com/qluvio/elv-mcp/tasks/tagstore"
)

// -----------------------------------------------------------------------------
// Tests — Validation & Error Contract
// -----------------------------------------------------------------------------

func TestTagStoreListTracks_MissingQID(t *testing.T) {
    cfg := &config.Config{}
    args := tagstore.TagStoreListTracksArgs{
        QID: "",
    }

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    ctx := runtime.WithTenant(context.Background(), NewMockTenant())

    res, payload, err := tagstore.TagStoreListTracksWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
    if err == nil {
        t.Fatalf("expected error for missing qid")
    }
    if !errors.IsKind(errors.K.Invalid, err) {
        t.Fatalf("expected Invalid error, got %v", err)
    }
    if res == nil || !res.IsError {
        t.Fatalf("CallToolResult must be non-nil and IsError=true on error")
    }
    if payload != nil {
        t.Fatalf("payload must be nil on error")
    }
}

func TestTagStoreListTracks_NoTenant(t *testing.T) {
    cfg := &config.Config{}
    args := tagstore.TagStoreListTracksArgs{
        QID: "iq__NO_TENANT",
    }

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    res, payload, err := tagstore.TagStoreListTracksWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
    if err == nil {
        t.Fatalf("expected error for missing tenant")
    }
    if !errors.IsKind(errors.K.Permission, err) {
        t.Fatalf("expected Permission error, got %v", err)
    }
    if res == nil || !res.IsError {
        t.Fatalf("CallToolResult must be non-nil and IsError=true on error")
    }
    if payload != nil {
        t.Fatalf("payload must be nil on error")
    }
}

// -----------------------------------------------------------------------------
// Tests — HTTP behavior
// -----------------------------------------------------------------------------

func TestTagStoreListTracks_Success(t *testing.T) {
    apiResp := struct {
        Tracks []tagstore.TagStoreTrack `json:"tracks"`
    }{
        Tracks: []tagstore.TagStoreTrack{
            {
                ID:          "id-1",
                QID:         "iq__SUCCESS",
                Name:        "speech",
                Label:       "Speech",
                Color:       "#FFFFFF",
                Description: "desc1",
            },
            {
                ID:          "id-2",
                QID:         "iq__SUCCESS",
                Name:        "chapters",
                Label:       "Chapters",
                Color:       "#000000",
                Description: "desc2",
            },
        },
    }

    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            t.Fatalf("expected GET, got %s", r.Method)
        }

        expected := "/tagstore/iq__SUCCESS/tracks"
        if r.URL.Path != expected {
            t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
        }

        w.Header().Set("Content-Type", "application/json")
        if err := json.NewEncoder(w).Encode(apiResp); err != nil {
            t.Fatalf("failed to encode response: %v", err)
        }
    }))
    defer srv.Close()

    cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
    args := tagstore.TagStoreListTracksArgs{
        QID: "iq__SUCCESS",
    }

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    ctx := runtime.WithTenant(context.Background(), NewMockTenant())

    res, payload, err := tagstore.TagStoreListTracksWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res == nil || res.IsError {
        t.Fatalf("expected non-error CallToolResult")
    }

    out, ok := payload.(*tagstore.TagStoreListTracksResult)
    if !ok {
        t.Fatalf("expected TagStoreListTracksResult, got %T", payload)
    }
    if out.QID != args.QID {
        t.Fatalf("unexpected QID: got %s, want %s", out.QID, args.QID)
    }
    if len(out.Tracks) != len(apiResp.Tracks) {
        t.Fatalf("unexpected tracks length: got %d, want %d", len(out.Tracks), len(apiResp.Tracks))
    }
    for i, tr := range apiResp.Tracks {
        got := out.Tracks[i]
        if got.ID != tr.ID || got.QID != tr.QID || got.Name != tr.Name ||
            got.Label != tr.Label || got.Color != tr.Color || got.Description != tr.Description {
            t.Fatalf("unexpected track at %d: got %+v, want %+v", i, got, tr)
        }
    }
}

func TestTagStoreListTracks_Unauthorized(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        expected := "/tagstore/iq__UNAUTH/tracks"
        if r.URL.Path != expected {
            t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
        }
        w.WriteHeader(http.StatusUnauthorized)
    }))
    defer srv.Close()

    cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
    args := tagstore.TagStoreListTracksArgs{
        QID: "iq__UNAUTH",
    }

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    ctx := runtime.WithTenant(context.Background(), NewMockTenant())

    res, payload, err := tagstore.TagStoreListTracksWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
    if err == nil {
        t.Fatalf("expected error for unauthorized")
    }
    if !errors.IsKind(errors.K.Permission, err) {
        t.Fatalf("expected Permission error, got %v", err)
    }
    if res == nil || !res.IsError {
        t.Fatalf("CallToolResult must be non-nil and IsError=true on error")
    }
    if payload != nil {
        t.Fatalf("payload must be nil on error")
    }
}

func TestTagStoreListTracks_NotFound(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        expected := "/tagstore/iq__NOT_FOUND/tracks"
        if r.URL.Path != expected {
            t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
        }
        w.WriteHeader(http.StatusNotFound)
    }))
    defer srv.Close()

    cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
    args := tagstore.TagStoreListTracksArgs{
        QID: "iq__NOT_FOUND",
    }

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    ctx := runtime.WithTenant(context.Background(), NewMockTenant())

    res, payload, err := tagstore.TagStoreListTracksWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
    if err == nil {
        t.Fatalf("expected error for not found")
    }
    if !errors.IsKind(errors.K.NotFound, err) {
        t.Fatalf("expected NotFound error, got %v", err)
    }
    if res == nil || !res.IsError {
        t.Fatalf("CallToolResult must be non-nil and IsError=true on error")
    }
    if payload != nil {
        t.Fatalf("payload must be nil on error")
    }
}

// -----------------------------------------------------------------------------
// Tests — MCP Error Contract
// -----------------------------------------------------------------------------

func TestTagStoreListTracks_MCPErrorContract(t *testing.T) {
    cfg := &config.Config{}

    prevAuth := auth.Auth
    auth.Auth = MockAuthProvider{}
    defer func() { auth.Auth = prevAuth }()

    res, payload, err := tagstore.TagStoreListTracksWorker(
        context.Background(),
        &mcp.CallToolRequest{},
        tagstore.TagStoreListTracksArgs{}, // invalid on purpose
        cfg,
    )

    if err == nil {
        t.Fatalf("expected error but got nil")
    }
    if res == nil {
        t.Fatalf("CallToolResult must not be nil on error")
    }
    if !res.IsError {
        t.Fatalf("IsError must be true on error")
    }
    if payload != nil {
        t.Fatalf("payload must be nil on error")
    }
}