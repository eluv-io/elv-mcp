package taggers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
)

func TestTaggerListModelsWorker_Success(t *testing.T) {
	// Mock /models endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
            "models": [
                {
                    "name": "celeb",
                    "description": "Celebrity Identification",
                    "type": "frame",
                    "tag_tracks": [
                        { "name": "celebrity_detection", "label": "Celebrity Detection" }
                    ]
                }
            ]
        }`))
	}))
	defer server.Close()

	cfg := &config.Config{
		AITaggerUrl: server.URL,
	}

	ctx := context.Background()

	res, payload, err := TaggerListModelsWorker(
		ctx,
		&mcp.CallToolRequest{},
		ListModelsArgs{}, // correct args type
		cfg,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("CallToolResult must not be nil")
	}

	out, ok := payload.(*ModelsResponse)
	if !ok {
		t.Fatalf("expected *ModelsResponse payload, got %T", payload)
	}

	if len(out.Models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(out.Models))
	}

	if out.Models[0].Name != "celeb" {
		t.Fatalf("unexpected model name: %s", out.Models[0].Name)
	}
}

func TestTaggerListModelsWorker_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	cfg := &config.Config{
		AITaggerUrl: server.URL,
	}

	ctx := context.Background()

	res, payload, err := TaggerListModelsWorker(
		ctx,
		&mcp.CallToolRequest{},
		ListModelsArgs{},
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

func TestTaggerListModelsWorker_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{ "models": INVALID_JSON }`))
	}))
	defer server.Close()

	cfg := &config.Config{
		AITaggerUrl: server.URL,
	}

	ctx := context.Background()

	res, payload, err := TaggerListModelsWorker(
		ctx,
		&mcp.CallToolRequest{},
		ListModelsArgs{},
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
