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
// Validation tests
// -----------------------------------------------------------------------------

func TestTagStoreCreateTrack_MissingQID(t *testing.T) {
	cfg := &config.Config{}
	args := tagstore.TagStoreCreateTrackArgs{
		QID:   "",
		Track: "speech",
	}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreCreateTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing qid")
	}
	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

func TestTagStoreCreateTrack_MissingTrack(t *testing.T) {
	cfg := &config.Config{}
	args := tagstore.TagStoreCreateTrackArgs{
		QID:   "iq__TEST",
		Track: "",
	}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreCreateTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing track")
	}
	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

func TestTagStoreCreateTrack_NoTenant(t *testing.T) {
	cfg := &config.Config{}
	args := tagstore.TagStoreCreateTrackArgs{
		QID:   "iq__NO_TENANT",
		Track: "speech",
	}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	res, payload, err := tagstore.TagStoreCreateTrackWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing tenant")
	}
	if !errors.IsKind(errors.K.Permission, err) {
		t.Fatalf("expected Permission error, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// HTTP behavior tests
// -----------------------------------------------------------------------------

func TestTagStoreCreateTrack_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		expected := "/tagstore/iq__SUCCESS/tracks/speech"
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"message":  "track created",
			"track_id": "track123",
		})
	}))
	defer srv.Close()

	cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
	args := tagstore.TagStoreCreateTrackArgs{
		QID:   "iq__SUCCESS",
		Track: "speech",
	}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreCreateTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil || res.IsError {
		t.Fatalf("expected non-error result")
	}

	out, ok := payload.(*tagstore.TagStoreCreateTrackResult)
	if !ok {
		t.Fatalf("expected TagStoreCreateTrackResult, got %T", payload)
	}
	if !out.Created || out.TrackID != "track123" {
		t.Fatalf("unexpected result: %+v", out)
	}
}

func TestTagStoreCreateTrack_BadRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		expected := "/tagstore/iq__BAD/tracks/speech"
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
		}

		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
	args := tagstore.TagStoreCreateTrackArgs{
		QID:   "iq__BAD",
		Track: "speech",
	}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreCreateTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for bad request")
	}
	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

func TestTagStoreCreateTrack_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		expected := "/tagstore/iq__UNAUTH/tracks/speech"
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
		}

		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
	args := tagstore.TagStoreCreateTrackArgs{
		QID:   "iq__UNAUTH",
		Track: "speech",
	}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreCreateTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for unauthorized")
	}
	if !errors.IsKind(errors.K.Permission, err) {
		t.Fatalf("expected Permission error, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

func TestTagStoreCreateTrack_Conflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		expected := "/tagstore/iq__CONFLICT/tracks/speech"
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: got %s, want %s", r.URL.Path, expected)
		}

		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	cfg := &config.Config{TagStoreUrl: srv.URL + "/tagstore"}
	args := tagstore.TagStoreCreateTrackArgs{
		QID:   "iq__CONFLICT",
		Track: "speech",
	}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	ctx := runtime.WithTenant(context.Background(), NewMockTenant())

	res, payload, err := tagstore.TagStoreCreateTrackWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if !errors.IsKind(errors.K.Exist, err) {
		t.Fatalf("expected Exists error, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
	if payload != nil {
		t.Fatalf("payload must be nil on error")
	}
}

// -----------------------------------------------------------------------------
// MCP Error Contract
// -----------------------------------------------------------------------------

func TestTagStoreCreateTrack_MCPErrorContract(t *testing.T) {
	cfg := &config.Config{}

	prev := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prev }()

	res, payload, err := tagstore.TagStoreCreateTrackWorker(
		context.Background(),
		&mcp.CallToolRequest{},
		tagstore.TagStoreCreateTrackArgs{}, // invalid on purpose
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
