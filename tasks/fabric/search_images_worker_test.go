package fabric_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
	"github.com/qluvio/elv-mcp/tasks/fabric"
)

//
// ────────────────────────────────────────────────────────────────
//   SECTION 1 — TEXT SEARCH PATH
// ────────────────────────────────────────────────────────────────
//

func TestSearchImagesWorker_TextSearch(t *testing.T) {
	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	// mock backend server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/search-ng/collections/iq__test_collection/search/text" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		io.WriteString(w, `{
            "results": [{
                "qid": "iq__abc",
                "similarity": 0.9,
                "match_info": {
                    "fps": "24000/1001",
                    "fps_float": 24.0,
                    "frame_idx": 48,
                    "offering": "default"
                }
            }]
        }`)
	}))
	defer srv.Close()

	cfg := &config.Config{
		SearchIdxUrl: srv.URL,
	}

	tf := &config.TenantFabric{
		SearchCollectionID: "iq__test_collection",
	}

	ctx := runtime.WithTenant(context.Background(), tf)

	args := fabric.SearchImagesArgs{
		Query: "chalkboard",
	}

	res, raw, err := fabric.SearchImagesWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}

	// Type‑assert the worker result
	imgRes, ok := raw.(*fabric.ImageSearchResult)
	if !ok {
		t.Fatalf("expected *ImageSearchResult, got %T", raw)
	}

	if imgRes == nil || len(imgRes.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(imgRes.Hits))
	}

	// Validate parsed hit
	h := imgRes.Hits[0]
	if h.QID != "iq__abc" {
		t.Fatalf("unexpected QID: %s", h.QID)
	}
	if h.Timestamp != 2.0 { // 48 / 24
		t.Fatalf("expected timestamp 2.0, got %f", h.Timestamp)
	}

	// Validate slim JSON
	tc := res.Content[0].(*mcp.TextContent)
	var slim fabric.SlimImageResponse
	if err := json.Unmarshal([]byte(tc.Text), &slim); err != nil {
		t.Fatalf("failed to decode slim JSON: %v", err)
	}

	if len(slim.Items) != 1 {
		t.Fatalf("expected 1 slim item, got %d", len(slim.Items))
	}
	if slim.Items[0].QID != "iq__abc" {
		t.Fatalf("unexpected slim QID: %s", slim.Items[0].QID)
	}
}

//
// ────────────────────────────────────────────────────────────────
//   SECTION 2 — IMAGE SEARCH PATH (multipart upload)
// ────────────────────────────────────────────────────────────────
//

func TestSearchImagesWorker_ImageSearch(t *testing.T) {
	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	// mock backend server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/search-ng/collections/iq__test_collection/search/image" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		// Validate multipart form
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart: %v", err)
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected file field: %v", err)
		}
		defer file.Close()

		// We don't care about content, just that it exists
		_, _ = io.ReadAll(file)

		io.WriteString(w, `{
            "results": [{
                "qid": "iq__img",
                "similarity": 0.8,
                "match_info": {
                    "fps": "24000/1001",
                    "fps_float": 24.0,
                    "frame_idx": 24,
                    "offering": "default"
                }
            }]
        }`)
	}))
	defer srv.Close()

	cfg := &config.Config{
		SearchIdxUrl: srv.URL,
	}

	tf := &config.TenantFabric{
		SearchCollectionID: "iq__test_collection",
	}

	ctx := runtime.WithTenant(context.Background(), tf)

	// Create a temporary fake image file
	tmp, err := os.CreateTemp("", "fake-image-*.jpg")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmp.Name())
	tmp.WriteString("fake image content")
	tmp.Close()

	args := fabric.SearchImagesArgs{
		ImagePath: tmp.Name(),
	}
	res, raw, err := fabric.SearchImagesWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}

	// Type‑assert the worker result
	imgRes, ok := raw.(*fabric.ImageSearchResult)
	if !ok {
		t.Fatalf("expected *ImageSearchResult, got %T", raw)
	}

	if imgRes == nil || len(imgRes.Hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(imgRes.Hits))
	}

}
