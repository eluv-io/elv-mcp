//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"maps"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/tasks/fabric"
)

// -----------------------------------------------------------------------------
// Integration Test — GetOfferingsWorker
// -----------------------------------------------------------------------------

func TestGetOfferingsWorker_Integration(t *testing.T) {
    cfg := loadIntegrationConfig(t)
    ctx := loadTenantContext(t, cfg, TagstoreTestTenant)

    args := fabric.GetOfferingsArgs{
        ContentID: IntegrationTestQID,
    }

    res, metaResp, err := fabric.GetOfferingsWorker(ctx, &mcp.CallToolRequest{}, args, cfg)
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

    out, ok := metaResp.(fabric.GetOfferingsResult)
    if !ok {
        t.Fatalf("expected GetOfferingsResult, got %T", metaResp)
    }

    // Print the metadata in a readable way
    jsonData, err := json.MarshalIndent(out.Offerings, "", "  ")
    if err != nil {
        t.Fatalf("failed to marshal metadata: %v", err)
    }

	offering_names := maps.Keys(out.Offerings)
	fmt.Printf("Offerings found: %v\n", offering_names)
    fmt.Println("GetOfferingsWorker returned metadata:")
    fmt.Println(string(jsonData))

    // Basic sanity checks
    // if name, ok := out.Offerings["name"]; ok && name == "" {
    //     t.Fatalf("expected non-empty name field")
    // }
}