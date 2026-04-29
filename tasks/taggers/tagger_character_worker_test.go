package taggers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eluv-io/errors-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
	async "github.com/qluvio/elv-mcp/tasks/async"
	"github.com/qluvio/elv-mcp/tasks/taggers"
)

//
// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

// newMockTenant and MockAuthProvider come from your existing test suite.
// They are assumed to be available in this package.

//
// -----------------------------------------------------------------------------
// Tests — Dependency Handling
// -----------------------------------------------------------------------------

func TestTagCharacters_MissingQID(t *testing.T) {
	cfg := &config.Config{}
	args := taggers.CharacterTaggingArgs{}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, _, err := taggers.TagCharactersWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing qid")
	}
	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
}

func TestTagCharacters_NoTenant(t *testing.T) {
	cfg := &config.Config{}
	args := taggers.CharacterTaggingArgs{
		QID: "iq__NO_TENANT",
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	_, _, err := taggers.TagCharactersWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing tenant")
	}
	if !errors.IsKind(errors.K.Permission, err) {
		t.Fatalf("expected Permission error, got %v", err)
	}
}

func TestTagCharacters_DependencyMissing_NoAutoRun(t *testing.T) {
	// Mock tag-status returning NO celeb model
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{
					"model":              "asr",
					"track":              "speech",
					"percent_completion": 1.0,
				},
			},
		})
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}
	args := taggers.CharacterTaggingArgs{
		QID: "iq__MISSING_CELEB",
		// AutoRunDependencies = false by default
		Synchronous: true,
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, _, err := taggers.TagCharactersWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected dependency error")
	}
	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
}

func TestTagCharacters_DependencyMissing_WithAutoRun(t *testing.T) {
	var startCalls atomic.Int32
	var statusCalls atomic.Int32

	qid := "iq__AUTO_RUN"

	// Mock Tagger server:
	// 1. tag-status → celeb missing
	// 2. POST /tag for celeb
	// 3. GET /job-status for celeb (running → completed)
	// 4. POST /tag for character
	// 5. GET /job-status for character (running → completed)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch {
		case r.URL.Path == "/"+qid+"/tag-status":
			// celeb missing
			json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{
					{
						"model":              "asr",
						"track":              "speech",
						"percent_completion": 1.0,
					},
				},
			})

		case r.URL.Path == "/"+qid+"/tag":
			// Called twice: first for celeb, then for character
			call := startCalls.Add(1)
			if call == 1 {
				// celeb start
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":  "job_celeb",
							"model":   "celeb",
							"stream":  "video",
							"started": true,
							"message": "started celeb",
						},
					},
				})
			} else {
				// character start
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":  "job_character",
							"model":   "character",
							"stream":  "video",
							"started": true,
							"message": "started character",
						},
					},
				})
			}

		case r.URL.Path == "/"+qid+"/job-status":
			call := statusCalls.Add(1)

			// First two calls → celeb job
			if call == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":           "job_celeb",
							"status":           "running",
							"time_running":     1.0,
							"tagging_progress": "0/1",
							"missing_tags":     []string{},
							"failed":           []string{},
							"model":            "celeb",
						},
					},
				})
				return
			}
			if call == 2 {
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":           "job_celeb",
							"status":           "completed",
							"time_running":     2.0,
							"tagging_progress": "1/1",
							"missing_tags":     []string{},
							"failed":           []string{},
							"model":            "celeb",
						},
					},
				})
				return
			}

			// Next two calls → character job
			if call == 3 {
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":           "job_character",
							"status":           "running",
							"time_running":     1.0,
							"tagging_progress": "0/1",
							"missing_tags":     []string{},
							"failed":           []string{},
							"model":            "character",
						},
					},
				})
				return
			}

			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":           "job_character",
						"status":           "completed",
						"time_running":     2.0,
						"tagging_progress": "1/1",
						"missing_tags":     []string{},
						"failed":           []string{},
						"model":            "character",
					},
				},
			})

		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}
	args := taggers.CharacterTaggingArgs{
		QID:                 qid,
		Synchronous:         true,
		AutoRunDependencies: true,
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, payload, err := taggers.TagCharactersWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out, ok := payload.(*taggers.CharacterTaggingSyncResult)
	if !ok {
		t.Fatalf("expected CharacterTaggingSyncResult, got %T", payload)
	}

	if len(out.AutoRanDependencies) != 1 || out.AutoRanDependencies[0] != "celeb" {
		t.Fatalf("expected auto-run celeb, got %v", out.AutoRanDependencies)
	}

	if len(out.Jobs) != 1 || out.Jobs[0].Model != "character" || out.Jobs[0].Status != "completed" {
		t.Fatalf("unexpected character job result: %+v", out.Jobs)
	}
}

//
// -----------------------------------------------------------------------------
// Tests — Async Mode
// -----------------------------------------------------------------------------

func TestTagCharacters_AsyncSuccess(t *testing.T) {
	var startCalls atomic.Int32
	var statusCalls atomic.Int32

	qid := "iq__ASYNC"

	// Mock Tagger server:
	// celeb already complete → no auto-run
	// character job runs async
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		switch {
		case r.URL.Path == "/"+qid+"/tag-status":
			json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{
					{
						"model":              "celeb",
						"track":              "celeb_track",
						"percent_completion": 1.0,
					},
				},
			})

		case r.URL.Path == "/"+qid+"/tag":
			startCalls.Add(1)
			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":  "job_character",
						"model":   "character",
						"stream":  "video",
						"started": true,
						"message": "started character",
					},
				},
			})

		case r.URL.Path == "/"+qid+"/job-status":
			call := statusCalls.Add(1)
			if call == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":           "job_character",
							"status":           "running",
							"time_running":     1.0,
							"tagging_progress": "0/1",
							"missing_tags":     []string{},
							"failed":           []string{},
							"model":            "character",
						},
					},
				})
				return
			}

			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":           "job_character",
						"status":           "completed",
						"time_running":     2.0,
						"tagging_progress": "1/1",
						"missing_tags":     []string{},
						"failed":           []string{},
						"model":            "character",
					},
				},
			})

		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := &config.Config{AITaggerUrl: srv.URL}
	args := taggers.CharacterTaggingArgs{
		QID:         qid,
		Synchronous: false,
	}

	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, payload, err := taggers.TagCharactersWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes, ok := payload.(*taggers.CharacterTaggingAsyncResult)
	if !ok {
		t.Fatalf("expected CharacterTaggingAsyncResult, got %T", payload)
	}
	if asyncRes.TaskID == "" {
		t.Fatalf("expected non-empty task ID")
	}

	// Wait for async task to complete
	deadline := time.Now().Add(3 * time.Second)
	for {
		snap, ok := async.GetSnapshot(asyncRes.TaskID)
		if !ok {
			t.Fatalf("task not found")
		}

		if snap.Status == async.StatusCompleted {
			break
		}
		if snap.Status == async.StatusFailed {
			t.Fatalf("task failed: %v", snap.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for async task")
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Validate final result
	snap, _ := async.GetSnapshot(asyncRes.TaskID)
	result := snap.Result["result"].(*taggers.CharacterTaggingSyncResult)

	if len(result.Jobs) != 1 || result.Jobs[0].Model != "character" {
		t.Fatalf("unexpected job result: %+v", result.Jobs)
	}
}

//
// -----------------------------------------------------------------------------
// Tests — MCP Error Contract
// -----------------------------------------------------------------------------

func TestTagCharacters_MCPErrorContract(t *testing.T) {
	cfg := &config.Config{}

	res, payload, err := taggers.TagCharactersWorker(
		context.Background(),
		&mcp.CallToolRequest{},
		taggers.CharacterTaggingArgs{}, // invalid on purpose
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
