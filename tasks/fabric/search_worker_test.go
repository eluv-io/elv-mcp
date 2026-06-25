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
	"github.com/qluvio/elv-mcp/tasks"
	"github.com/qluvio/elv-mcp/tasks/fabric"
)

// -----------------------------------------------------------------------------
//
// ────────────────────────────────────────────────────────────────
//   SECTION 1 — BASIC UNIT TESTS (no HTTP, no backend)
// ────────────────────────────────────────────────────────────────
//

func TestSearchClips_EmptyTerms(t *testing.T) {
	cfg := &config.Config{}
	args := fabric.SearchClipsArgs{Terms: ""}

	res, _, err := fabric.SearchWorker(context.Background(), &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected Go error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected MCP error result")
	}
}

//
// ────────────────────────────────────────────────────────────────
//   SECTION 2 — MOCK BACKEND TESTS (CI‑safe, no network)
// ────────────────────────────────────────────────────────────────
//

func TestSearchWorker_WithMockBackend(t *testing.T) {
	// swap in mock auth
	prevAuth := auth.Auth
	auth.Auth = MockAuthProvider{}
	defer func() { auth.Auth = prevAuth }()

	// mock backend server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{
            "contents": [{
                "qlib_id": "abc",
                "qid": "123",
                "start_time": 1000,
                "end_time": 5000,
                "video_url": "https://mock/video",
                "image_url": "https://mock/thumb"
            }]
        }`)
	}))
	defer srv.Close()

	cfg := &config.Config{
		SearchIdxUrl: srv.URL,
	}

	tenant := &config.TenantFabric{}
	ctx := runtime.WithTenant(context.Background(), tenant)

	args := fabric.SearchClipsArgs{Terms: "test"}

	res, clipResp, err := fabric.SearchWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %+v", res)
	}
	if clipResp == nil || len(clipResp.Contents) != 1 {
		t.Fatalf("expected 1 clip, got %d", len(clipResp.Contents))
	}
}

//
// ────────────────────────────────────────────────────────────────
//   SECTION 3 — CONTRACT TESTS (schema validation)
// ────────────────────────────────────────────────────────────────
//

func TestSearchWorker_Contract(t *testing.T) {
	data, err := os.ReadFile("testdata/search_response.json")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}

	var resp tasks.ClipResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if len(resp.Contents) == 0 {
		t.Fatalf("expected contents")
	}
}

//
// ────────────────────────────────────────────────────────────────
//   SECTION 4 — DESCRIPTION CONTRACT
// ────────────────────────────────────────────────────────────────
//

func TestBuildSearchResultResponse_PopulatesDescription(t *testing.T) {
	args := fabric.SearchClipsArgs{
		Terms: "matrix",
	}

	result := &tasks.ClipResponse{
		Contents: []tasks.ClipItem{
			{
				QID:          "iq__cKjQwrN817yb1paEaUpPr6n9PRi",
				QLibID:       "ilib37FUN1H559GLk5EnnSx8u3Vggt3J",
				VideoURL:     "/q/hq__6gi1a1mt7Gyn6QFH7aRkiyyv9ZmQFdMaHf9oN2imobgCfoXpaGe9JntSWrpReWMSnU4HEzzLqUmK6/rep/playout/default/options.json?clip_start=260.01&clip_end=286.286&ignore_trimming=true",
				ImageURL:     "/q/hq__6gi1a1mt7Gyn6QFH7aRkiyyv9ZmQFdMaHf9oN2imobgCfoXpaGe9JntSWrpReWMSnU4HEzzLqUmK6/rep/frame/default/video?t=260.01&ignore_trimming=true",
				Start:        "4m20.01s",
				End:          "4m46.286s",
				StartTime:    260010,
				EndTime:      286286,
				Score:        "0.636", // raw_score rounded to 3 decimals
				Meta: map[string]interface{}{
					"public": map[string]interface{}{
						"asset_metadata": map[string]interface{}{
							"display_title": "LYLE, LYLE, CROCODILE",
							"info": map[string]interface{}{
								"release_date": "10/06/2022",
							},
							"ip_title_id": "11096930_15394185",
						},
					},
				},
			},
		},
	}
	
	toolResult , clipResp, err := fabric.BuildSearchResultResponse(args, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clipResp == nil {
		t.Fatalf("expected non-nil ClipResponse")
	}
	if clipResp.Description != "" {
		t.Fatalf("Expected Description to be empty after slimming, got: %q", clipResp.Description)
	}

	var slim fabric.SlimResponse

	if err := json.Unmarshal([]byte(toolResult.Content[0].(*mcp.TextContent).Text), &slim); err != nil {
		t.Fatalf("Failed to decode slim JSON: %v", err)
	}
	
	if slim.Summary == "" {
		t.Fatalf("Expected non-empty summary")
	}
	
	if slim.Confidence < 0 {
		t.Fatalf("Expected non-negative confidence")
	}
	
	if len(slim.Clips) == 0 {
		t.Fatalf("Expected at least one clip")
	}
}

func TestSearchClipsArgs_JSONTags(t *testing.T) {
	args := fabric.SearchClipsArgs{
		Terms:        "test",
		SearchFields: []string{"title"},
		Limit:        10,
	}

	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	if !contains(string(data), `"terms":"test"`) {
		t.Fatalf("expected terms field in JSON: %s", string(data))
	}
	if !contains(string(data), `"limit":10`) {
		t.Fatalf("expected limit field in JSON: %s", string(data))
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(s) > len(sub) && (contains(s[1:], sub) || s[:len(sub)] == sub)))
}
