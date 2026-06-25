package tagstore_test

import (
	"context"
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

func TestTagStoreDeleteTrack_MissingQID(t *testing.T) {
	cfg := &config.Config{}
	args := tagstore.TagStoreDeleteTrackArgs{
		QID:   "",
		Track: "speech",
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreDeleteTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
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

func TestTagStoreDeleteTrack_MissingTrack(t *testing.T) {
	cfg := &config.Config{}
	args := tagstore.TagStoreDeleteTrackArgs{
		QID:   "iq__TEST",
		Track: "",
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreDeleteTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing track")
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

func TestTagStoreDeleteTrack_NoTenant(t *testing.T) {
	cfg := &config.Config{}
	args := tagstore.TagStoreDeleteTrackArgs{
		QID:   "iq__NO_TENANT",
		Track: "speech",
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	res, payload, err := tagstore.TagStoreDeleteTrackWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
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

func TestTagStoreDeleteTrack_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}

		expected := "/tagstore/iq__SUCCESS/tracks/speech"
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
	args := tagstore.TagStoreDeleteTrackArgs{
		QID:   "iq__SUCCESS",
		Track: "speech",
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreDeleteTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.IsError {
		t.Fatalf("expected non-error CallToolResult")
	}

	out, ok := payload.(*tagstore.TagStoreDeleteTrackResult)
	if !ok {
		t.Fatalf("expected TagStoreDeleteTrackResult, got %T", payload)
	}
	if !out.Deleted || out.QID != args.QID || out.Track != args.Track {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestTagStoreDeleteTrack_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		expected := "/tagstore/iq__UNAUTH/tracks/speech"
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
		}

		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
	args := tagstore.TagStoreDeleteTrackArgs{
		QID:   "iq__UNAUTH",
		Track: "speech",
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreDeleteTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
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

func TestTagStoreDeleteTrack_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		expected := "/tagstore/iq__NOT_FOUND/tracks/speech"
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
	args := tagstore.TagStoreDeleteTrackArgs{
		QID:   "iq__NOT_FOUND",
		Track: "speech",
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreDeleteTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
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

func TestTagStoreDeleteTrack_MCPErrorContract(t *testing.T) {
	cfg := &config.Config{}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	res, payload, err := tagstore.TagStoreDeleteTrackWorker(
		context.Background(),
		&mcp.CallToolRequest{},
		tagstore.TagStoreDeleteTrackArgs{}, // invalid on purpose
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
