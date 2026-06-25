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

// -----------------------------------------------------------------------------
// Integration Test — GetPublicMetaWorker
// -----------------------------------------------------------------------------

func TestGetPublicMetaWorker_Integration(t *testing.T) {
    cfg := loadIntegrationConfig(t)
    ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

    args := fabric.GetPublicMetaArgs{
        ContentID: IntegrationTestQID,
    }

    res, metaResp, err := fabric.GetPublicMetaWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if res == nil {
        t.Fatalf("expected non-nil CallToolResult")
    }
    if res.IsError {
        t.Fatalf("tool returned error: %+v", res)
    }
    if metaResp == nil {
        t.Fatalf("expected non-nil metadata response")
    }

    out, ok := metaResp.(fabric.GetPublicMetaResult)
    if !ok {
        t.Fatalf("expected GetPublicMetaResult, got %T", metaResp)
    }

    // Print the metadata in a readable way
    jsonData, err := json.MarshalIndent(out.Data, "", "  ")
    if err != nil {
        t.Fatalf("failed to marshal metadata: %v", err)
    }

    fmt.Println("GetPublicMetaWorker returned metadata:")
    fmt.Println(string(jsonData))

    // Basic sanity checks
    if name, ok := out.Data["name"]; ok && name == "" {
        t.Fatalf("expected non-empty name field")
    }
}
