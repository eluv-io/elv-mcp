package taggers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/eluv-io/errors-go"
	"github.com/eluv-io/log-go"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

func TaggerListModelsWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args ListModelsArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// ---------------------------------------------------------
	// Tenant lookup
	// ---------------------------------------------------------
	// tf, ok := runtime.TenantFromContext(ctx)
	// if !ok {
	//     return runtime.MCPError(
	//         errors.E("list_models", errors.K.Permission,
	//             "reason", "tenant not found in context"),
	//     )
	// }

	// ---------------------------------------------------------
	// Build request
	// ---------------------------------------------------------
	baseURL := cfg.AITaggerUrl
	url := baseURL + "/models"

	log.Debug("Listing Models", "Request", url)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return runtime.MCPError(
			errors.E("list_models", errors.K.Invalid,
				"reason", "failed to build request", "error", err),
		)
	}

	// ---------------------------------------------------------
	// Execute request
	// ---------------------------------------------------------
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return runtime.MCPError(
			errors.E("list_models", errors.K.Unavailable,
				"reason", "request failed", "error", err),
		)
	}

	log.Debug("Listing Models", "Response", resp)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	log.Debug("Listing Models", "Response Body", string(body))
	if err != nil {
		return runtime.MCPError(
			errors.E("list_models", errors.K.Invalid,
				"reason", "failed to read response", "error", err),
		)
	}

	// ---------------------------------------------------------
	// Non‑200 response
	// ---------------------------------------------------------
	if resp.StatusCode != http.StatusOK {
		return runtime.MCPError(
			errors.E("list_models", errors.K.Unavailable,
				"reason", fmt.Sprintf("unexpected status %d from /models", resp.StatusCode),
				"body", string(body)),
		)
	}

	// ---------------------------------------------------------
	// Decode JSON
	// ---------------------------------------------------------
	var result ModelsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return runtime.MCPError(
			errors.E("list_models", errors.K.Invalid,
				"reason", "invalid JSON in /models response", "error", err),
		)
	}

	// ---------------------------------------------------------
	// Success
	// ---------------------------------------------------------
	return &mcp.CallToolResult{}, &result, nil
}
