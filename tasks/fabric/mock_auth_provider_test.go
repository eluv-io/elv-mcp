package fabric_test

import (
	"github.com/qluvio/elv-mcp/config"
)

// -----------------------------------------------------------------------------
// Mock AuthProvider
// -----------------------------------------------------------------------------

//
// ────────────────────────────────────────────────────────────────
//   INFRASTRUCTURE — Mock repository
// ────────────────────────────────────────────────────────────────
//

type MockAuthProvider struct{}

func (MockAuthProvider) FetchStateChannel(cfg *config.Config, tf *config.TenantFabric) (string, error) {
	return "dummy-state-token", nil
}

func (MockAuthProvider) FetchEditorSigned(cfg *config.Config, tf *config.TenantFabric, qlibID, qid string) (string, error) {
	return "dummy-editor-token", nil
}

func (MockAuthProvider) GetQLibId(cfg *config.Config, tf *config.TenantFabric, QID string) (string, string, error) {
	return "qlibid", "", nil
}


// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func newMockTenant() *config.TenantFabric {
	return &config.TenantFabric{
		PkStr:       "0x123",
		QLibIndexID: "qlibid",
		QIndexID:    "qid",
	}
}
