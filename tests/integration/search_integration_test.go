//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/tasks/fabric"
)

func TestSearchWorker_Integration(t *testing.T) {
	cfg := loadIntegrationConfig(t)
	ctx := loadTenantContext(t, cfg, "urc-content-ops")

	// Andrea - This test uses URC
	args := fabric.SearchClipsArgs{
		Terms: "best kick",
	}

	res, clipResp, err := fabric.SearchWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res)
	}
	if clipResp == nil || len(clipResp.Contents) == 0 {
		t.Fatalf("expected at least one clip")
	}

	if clipResp == nil || clipResp.Description != "" {
		t.Fatalf("expected non-empty description")
	}

	if res == nil || res.Content[0] == nil {
		t.Fatalf("expected non-empty content text")
	}

	fmt.Println("SearchWorker returned:", res.Content[0].(*mcp.TextContent).Text)


	// Use json.MarshalIndent and handle both return values
	jsonData, err := json.MarshalIndent(clipResp.Contents, "", " ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	println("SearchWorker content :", string(jsonData))
}
