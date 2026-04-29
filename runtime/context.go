package runtime

import (
	"context"

	"github.com/qluvio/elv-mcp/config"
)

type tenantContextKey struct{}

// TenantFromContext retrieves the TenantFabric stored by selectiveAuthMiddleware.
func TenantFromContext(ctx context.Context) (*config.TenantFabric, bool) {
	tf, ok := ctx.Value(tenantContextKey{}).(*config.TenantFabric)
	return tf, ok
}

// Exported so mcpserver can set the tenant in context.
func WithTenant(ctx context.Context, tf *config.TenantFabric) context.Context {
	return context.WithValue(ctx, tenantContextKey{}, tf)
}
