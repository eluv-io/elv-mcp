package auth

import "github.com/qluvio/elv-mcp/config"

// Provider abstracts token fetching for all tasks (fabric, taggers, etc.).
type Provider interface {
    FetchStateChannel(cfg *config.Config, tf *config.TenantFabric) (string, error)
    FetchEditorSigned(cfg *config.Config, tf *config.TenantFabric, qlibID, qid string) (string, error)
    GetQLibId(cfg *config.Config, tf *config.TenantFabric, QID string) (string, string, error)
}

// Auth is the global provider used by all tasks.
// In production, this is set to realAuthProvider in auth.go.
var Auth Provider
