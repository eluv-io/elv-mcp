package taggers_test

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
	"github.com/qluvio/elv-mcp/tasks/taggers"
)

// -----------------------------------------------------------------------------
// Summary Mode
// -----------------------------------------------------------------------------

func TestTaggerTagStatusWorker_Summary(t *testing.T) {
	// Stub auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__SUMMARY_TEST"

	// Mock tagger server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+qid+"/tag-status" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("authorization") != "dummy-editor-token" {
			t.Fatalf("expected authorization=dummy-editor-token, got %s", r.URL.Query().Get("authorization"))
		}

		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"model":              "celeb",
					"track":              "celebrity_detection",
					"last_run":           "2026-02-25T18:30:00Z",
					"percent_completion": 0.85,
				},
			},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}
	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	res, payload, err := taggers.TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.TaggerTagStatusArgs{QID: qid},
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}

	out, ok := payload.(taggers.TagStatusSummaryResponse)
	if !ok {
		t.Fatalf("expected []TagStatusSummaryResponse, got %T", payload)
	}

	if len(out.Statuses) != 1 {
		t.Fatalf("expected 1 model summary, got %d", len(out.Statuses))
	}
	if out.Statuses[0].Model != "celeb" {
		t.Fatalf("unexpected model: %s", out.Statuses[0].Model)
	}
}

// -----------------------------------------------------------------------------
// Model Detail Mode
// -----------------------------------------------------------------------------

func TestTaggerTagStatusWorker_ModelDetail(t *testing.T) {
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__MODEL_TEST"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+qid+"/tag-status/celeb" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("authorization") != "dummy-editor-token" {
			t.Fatalf("expected authorization=dummy-editor-token")
		}

		json.NewEncoder(w).Encode(map[string]any{
			"summary": map[string]any{
				"model":             "celeb",
				"track":             "celebrity_detection",
				"last_run":          "2026-02-25T18:30:00Z",
				"tagging_progress":  0.85,
				"num_content_parts": 120,
			},
			"jobs": []map[string]any{
				{
					"time_ran":      "1h0m0s",
					"source_qid":    "iq__123",
					"params":        map[string]any{"fps": 2},
					"job_status":    map[string]any{"status": "Completed"},
					"upload_status": map[string]any{"num_tagged_parts": 60},
				},
			},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}
	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	res, payload, err := taggers.TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.TaggerTagStatusArgs{QID: qid, Model: "celeb"},
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}

	out, ok := payload.(taggers.TagStatusModelDetail)
	if !ok {
		t.Fatalf("expected TagStatusModelDetail, got %T", payload)
	}

	if out.Summary.Model != "celeb" {
		t.Fatalf("unexpected model: %s", out.Summary.Model)
	}
	if len(out.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(out.Jobs))
	}
}

// -----------------------------------------------------------------------------
// Non‑200 Response
// -----------------------------------------------------------------------------

func TestTaggerTagStatusWorker_Non200(t *testing.T) {
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__BAD_STATUS"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}
	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	res, payload, err := taggers.TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.TaggerTagStatusArgs{QID: qid},
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

// -----------------------------------------------------------------------------
// Missing QID
// -----------------------------------------------------------------------------

func TestTaggerTagStatusWorker_MissingQID(t *testing.T) {
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	cfg := &config.Config{}
	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, _, err := taggers.TaggerTagStatusWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.TaggerTagStatusArgs{},
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error for missing qid")
	}
	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// Missing Tenant
// -----------------------------------------------------------------------------

func TestTaggerTagStatusWorker_NoTenant(t *testing.T) {
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	cfg := &config.Config{}

	_, _, err := taggers.TaggerTagStatusWorker(
		context.Background(),
		&mcp.CallToolRequest{},
		taggers.TaggerTagStatusArgs{QID: "iq__NO_TENANT"},
		cfg,
	)

	if err == nil {
		t.Fatalf("expected error for missing tenant")
	}
	if !errors.IsKind(errors.K.Permission, err) {
		t.Fatalf("expected Permission error, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// MCP Error Contract
// -----------------------------------------------------------------------------

func TestTaggerTagStatusWorker_MCPErrorContract(t *testing.T) {
	cfg := &config.Config{}

	res, payload, err := taggers.TaggerTagStatusWorker(
		context.Background(),
		&mcp.CallToolRequest{},
		taggers.TaggerTagStatusArgs{}, // invalid on purpose
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
