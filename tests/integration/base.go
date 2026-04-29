//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/mcpserver"
	"github.com/qluvio/elv-mcp/runtime"
	"github.com/qluvio/elv-mcp/tasks/tagstore"
)

// -----------------------------------------------------------------------------
// Shared test constants
// -----------------------------------------------------------------------------

// Stable content object for integration testing
const TagstoreTestTenant = "urc-content-ops"
const IntegrationTestQID = "iq__47cbSU6ygSF5Zaoc6RfCyS4E1Ppr"

// loadIntegrationConfig loads the config file specified by ELV_MCP_CONFIG.
// If the environment variable is not set, the test is skipped.
func loadIntegrationConfig(t *testing.T) *config.Config {
	cfgPath := os.Getenv("ELV_MCP_CONFIG")
	if cfgPath == "" {
		t.Skip("ELV_MCP_CONFIG not set; skipping integration test")
	}

	cfg, err := config.LoadConfigWithPath(cfgPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	return cfg
}

// loadTenantContext loads a tenant from the config and returns a context
// scoped to that tenant's Fabric configuration.
func loadTenantContext(t *testing.T, cfg *config.Config, sub string) context.Context {
	tenant, ok := cfg.Tenants.Lookup(sub)
	if !ok {
		t.Fatalf("tenant not found for sub %q", sub)
	}
	return runtime.WithTenant(context.Background(), tenant.Fabric)
}

// newTestServer constructs a real MCP server using the provided config.
func newTestServer(t *testing.T, cfg *config.Config) *mcp.Server {
	server := mcpserver.NewServer(cfg)
	if server == nil {
		t.Fatalf("failed to create MCP server")
	}
	return server
}

// deleteTracksBestEffort attempts to delete the specified tracks, but does not fail the test if deletion fails.
// used to clean up tracks created during tests, without masking the original test failure if deletion fails.
func deleteTracksBestEffort(t *testing.T, ctx context.Context, cfg *config.Config, tracks ...string) {
	t.Helper()
	for _, tr := range tracks {
		_, _, _ = tagstore.TagStoreDeleteTrackWorker(
			ctx,
			&mcp.CallToolRequest{},
			tagstore.TagStoreDeleteTrackArgs{
				QID:   IntegrationTestQID,
				Track: tr,
			},
			cfg,
		)
	}
}
