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
// Tests — Stop All Jobs
// -----------------------------------------------------------------------------

func TestStopTagging_Sync_AllJobs(t *testing.T) {
	// Use shared mock auth provider
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__STOP_SYNC_ALL"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+qid+"/stop" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jobs": []map[string]any{
				{"job_id": "job1", "message": "stopped"},
				{"job_id": "job2", "message": "stopped"},
			},
			"message": "all stopped",
		})
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}

	args := taggers.TaggerStopArgs{
		QID: qid,
	}

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, result, err := taggers.TaggerStopWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	syncRes := result.(*taggers.TaggerStopResult)
	if len(syncRes.Jobs) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(syncRes.Jobs))
	}

	if syncRes.Jobs[0].JobID == "" {
		t.Fatalf("expected non-empty job_id")
	}
	if syncRes.Jobs[0].Message == "" {
		t.Fatalf("expected non-empty message")
	}
}

// -----------------------------------------------------------------------------
// Tests — Stop Specific Model
// -----------------------------------------------------------------------------

func TestStopTagging_Sync_Model(t *testing.T) {
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__STOP_SYNC_MODEL"
	model := "asr"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/" + qid + "/stop/" + model
		if r.URL.Path != expected {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"jobs": []map[string]any{
				{"job_id": "job1", "message": "stopped"},
			},
			"message": "model stopped",
		})
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}

	args := taggers.TaggerStopArgs{
		QID:   qid,
		Model: model,
	}

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, result, err := taggers.TaggerStopWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	syncRes := result.(*taggers.TaggerStopResult)
	if len(syncRes.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(syncRes.Jobs))
	}

	if syncRes.Jobs[0].JobID == "" {
		t.Fatalf("expected non-empty job_id")
	}
	if syncRes.Jobs[0].Message == "" {
		t.Fatalf("expected non-empty message")
	}
}

// -----------------------------------------------------------------------------
// Tests — Missing QID
// -----------------------------------------------------------------------------

func TestStopTagging_MissingQID(t *testing.T) {
	cfg := &config.Config{}
	args := taggers.TaggerStopArgs{}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, _, err := taggers.TaggerStopWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing qid")
	}

	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// Tests — Missing Tenant
// -----------------------------------------------------------------------------

func TestStopTagging_NoTenant(t *testing.T) {
	cfg := &config.Config{}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	args := taggers.TaggerStopArgs{
		QID: "iq__STOP_NO_TENANT",
	}

	_, _, err := taggers.TaggerStopWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing tenant")
	}

	if !errors.IsKind(errors.K.Permission, err) {
		t.Fatalf("expected Permission error, got %v", err)
	}
}

// -----------------------------------------------------------------------------
// Tests — Error Handling Contract
// -----------------------------------------------------------------------------

func TestTaggerStopWorker_MCPErrorContract(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}

	res, payload, err := taggers.TaggerStopWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.TaggerStopArgs{}, // invalid on purpose
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
