// tagger_start_handler_test.go
package taggers_test

import (
	"context"
	"encoding/json"
	"log"
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

// -----------------------------------------------------------------------------
// Tests
// -----------------------------------------------------------------------------

func TestTagContent_SyncSuccess(t *testing.T) {
	var statusCalls atomic.Int32
	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__AAAAAAAnonymizedSync"

	// Mock Tagger server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + qid + "/tag":
			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":  "job1",
						"model":   "asr",
						"stream":  "audio_1",
						"started": true,
						"message": "successfully started",
						"error":   nil,
					},
				},
			})
		case "/" + qid + "/job-status":
			// First call: running
			if statusCalls.Add(1) == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":           "job1",
							"status":           "running",
							"time_running":     1.0,
							"tagging_progress": "0/1",
							"missing_tags":     []string{},
							"failed":           []string{},
							"model":            "asr",
							"stream":           "audio_1",
						},
					},
				})
				return
			}
			// Second call: completed
			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":           "job1",
						"status":           "completed",
						"time_running":     2.0,
						"tagging_progress": "1/1",
						"missing_tags":     []string{},
						"failed":           []string{},
						"model":            "asr",
						"stream":           "audio_1",
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := &config.Config{
		AITaggerUrl: srv.URL,
	}

	args := taggers.TagContentArgs{
		QID:         qid,
		Synchronous: true,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
		},
	}

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, result, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	syncRes, ok := result.(*taggers.TagContentSyncResult)
	if !ok {
		t.Fatalf("expected TagContentSyncResult, got %T", result)
	}

	if len(syncRes.Jobs) != 1 {
		t.Fatalf("expected 1 job, got %d", len(syncRes.Jobs))
	}

	if syncRes.Jobs[0].Status != "completed" {
		t.Fatalf("expected completed, got %s", syncRes.Jobs[0].Status)
	}
}

func TestTagContent_AsyncSuccess(t *testing.T) {
	var statusCalls atomic.Int32

	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__BBBBBBAnonymizedAsync"

	// Mock Tagger server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		switch r.URL.Path {
		case "/" + qid + "/tag":
			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":  "job1",
						"model":   "asr",
						"stream":  "audio_1",
						"started": true,
						"message": "successfully started",
						"error":   nil,
					},
				},
			})
		case "/" + qid + "/job-status":
			if statusCalls.Add(1) == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":           "job1",
							"status":           "running",
							"time_running":     1.0,
							"tagging_progress": "0/1",
							"missing_tags":     []string{},
							"failed":           []string{},
							"model":            "asr",
							"stream":           "audio_1",
						},
					},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":           "job1",
						"status":           "completed",
						"time_running":     2.0,
						"tagging_progress": "1/1",
						"missing_tags":     []string{},
						"failed":           []string{},
						"model":            "asr",
						"stream":           "audio_1",
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := &config.Config{
		AITaggerUrl: srv.URL,
	}

	args := taggers.TagContentArgs{
		QID:         qid,
		Synchronous: false,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
		},
	}

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, result, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes, ok := result.(*taggers.TagContentAsyncResult)
	if !ok {
		t.Fatalf("expected TagContentAsyncResult, got %T", result)
	}

	if asyncRes.TaskID == "" {
		t.Fatalf("expected task ID")
	}

	// Wait for async task to complete
	time.Sleep(200 * time.Millisecond)

	snap, ok := async.GetSnapshot(asyncRes.TaskID)
	if !ok {
		t.Fatalf("task not found")
	}

	// Wait for async task to complete
	time.Sleep(200 * time.Millisecond)

	deadline := time.Now().Add(3 * time.Second)

	for {
		snap, ok = async.GetSnapshot(asyncRes.TaskID)
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
			t.Fatalf("timeout waiting for task to complete, last status=%s", snap.Status)
		}

		time.Sleep(20 * time.Millisecond)
	}

	jobs := snap.Result["result"].([]taggers.TagJobStatus)
	if len(jobs) != 1 || jobs[0].Status != "completed" {
		t.Fatalf("unexpected job result: %+v", jobs)
	}
}

func TestTagContent_AsyncSuccessWithModels(t *testing.T) {
	var statusCalls atomic.Int32

	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	qid := "iq__BBBBBBAnonymizedAsync"

	// Mock Tagger server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request: %s %s", r.Method, r.URL.Path)
		switch r.URL.Path {
		case "/" + qid + "/tag":
			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":  "job1",
						"model":   "asr",
						"stream":  "audio_1",
						"started": true,
						"message": "successfully started",
						"error":   nil,
					},
				},
			})
		case "/" + qid + "/job-status":
			if statusCalls.Add(1) == 1 {
				json.NewEncoder(w).Encode(map[string]any{
					"jobs": []map[string]any{
						{
							"job_id":           "job1",
							"status":           "running",
							"time_running":     1.0,
							"tagging_progress": "0/1",
							"missing_tags":     []string{},
							"failed":           []string{},
							"model":            "asr",
							"stream":           "audio_1",
						},
					},
				})
				return
			}
			json.NewEncoder(w).Encode(map[string]any{
				"jobs": []map[string]any{
					{
						"job_id":           "job1",
						"status":           "completed",
						"time_running":     2.0,
						"tagging_progress": "1/1",
						"missing_tags":     []string{},
						"failed":           []string{},
						"model":            "asr",
						"stream":           "audio_1",
					},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	cfg := &config.Config{
		AITaggerUrl: srv.URL,
	}

	args := taggers.TagContentArgs{
		QID:         qid,
		Synchronous: false,
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
			{Model: "ocr"},
		},
	}

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, result, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	asyncRes, ok := result.(*taggers.TagContentAsyncResult)
	if !ok {
		t.Fatalf("expected TagContentAsyncResult, got %T", result)
	}

	if asyncRes.TaskID == "" {
		t.Fatalf("expected task ID")
	}

	// Wait for async task to complete
	time.Sleep(200 * time.Millisecond)

	snap, ok := async.GetSnapshot(asyncRes.TaskID)
	if !ok {
		t.Fatalf("task not found")
	}

	// Wait for async task to complete
	time.Sleep(200 * time.Millisecond)

	deadline := time.Now().Add(3 * time.Second)

	for {
		snap, ok = async.GetSnapshot(asyncRes.TaskID)
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
			t.Fatalf("timeout waiting for task to complete, last status=%s", snap.Status)
		}

		time.Sleep(20 * time.Millisecond)
	}

	jobs := snap.Result["result"].([]taggers.TagJobStatus)
	if len(jobs) != 1 || jobs[0].Status != "completed" {
		t.Fatalf("unexpected job result: %+v", jobs)
	}
}

func TestTagContent_MissingQID(t *testing.T) {
	cfg := &config.Config{}
	args := taggers.TagContentArgs{}
	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	ctx := runtime.WithTenant(context.Background(), newMockTenant())

	_, _, err := taggers.TagContentWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err == nil {
		t.Fatalf("expected error for missing qid")
	}

	if !errors.IsKind(errors.K.Invalid, err) {
		t.Fatalf("expected Invalid error, got %v", err)
	}
}

func TestTagContent_NoTenant(t *testing.T) {
	cfg := &config.Config{}
	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	args := taggers.TagContentArgs{
		QID: "iq__CCCCCCNoTenant",
		Jobs: []taggers.TagJobSpec{
			{Model: "asr"},
		},
	}

	_, _, err := taggers.TagContentWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
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

func TestTagContentWorker_MCPErrorContract(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{}

	// Force an error by passing empty args
	res, payload, err := taggers.TagContentWorker(
		ctx,
		&mcp.CallToolRequest{},
		taggers.TagContentArgs{}, // invalid on purpose
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
