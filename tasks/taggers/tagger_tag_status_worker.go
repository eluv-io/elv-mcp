package taggers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/eluv-io/errors-go"
	"github.com/eluv-io/log-go"
	auth "github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

func TaggerTagStatusWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args TaggerTagStatusArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// -------------------------------------------------------------------------
	// Validate input
	// -------------------------------------------------------------------------
	if args.QID == "" {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Invalid, "reason", "qid is required"),
		)
	}

	// -------------------------------------------------------------------------
	// Resolve tenant (required for authorization token)
	// -------------------------------------------------------------------------
	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Permission,
				"reason", "tenant not found in context"),
		)
	}

	// -------------------------------------------------------------------------
	// Retrieve authorization token for this QID
	// -------------------------------------------------------------------------
	token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", args.QID)
	if err != nil {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Permission,
				"reason", "failed to retrieve authorization token", "error", err),
		)
	}

	// -------------------------------------------------------------------------
	// Build request URL
	// -------------------------------------------------------------------------
	base := strings.TrimRight(cfg.AITaggerUrl, "/")

	var url string
	if args.Model == "" {
		url = fmt.Sprintf("%s/%s/tag-status?authorization=%s", base, args.QID, token)
	} else {
		url = fmt.Sprintf("%s/%s/tag-status/%s?authorization=%s", base, args.QID, args.Model, token)
	}

	log.Debug("Tag Status", "Request", url)

	// -------------------------------------------------------------------------
	// Build HTTP request
	// -------------------------------------------------------------------------
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Invalid,
				"reason", "failed to build request", "error", err),
		)
	}

	// -------------------------------------------------------------------------
	// Execute HTTP request
	// -------------------------------------------------------------------------
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Unavailable,
				"reason", "request failed", "error", err),
		)
	}

	log.Debug("Tag Status", "Response", resp)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Invalid,
				"reason", "failed to read response", "error", err),
		)
	}

	log.Debug("Tag Status", "Response Body", string(body))

	// -------------------------------------------------------------------------
	// Handle non‑200 responses
	// -------------------------------------------------------------------------
	if resp.StatusCode != http.StatusOK {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Unavailable,
				"reason", fmt.Sprintf("unexpected status %d", resp.StatusCode),
				"body", string(body)),
		)
	}

	// -------------------------------------------------------------------------
	// Decode + Normalize
	// -------------------------------------------------------------------------
	if args.Model == "" {
		// -----------------------------
		// Summary mode
		// -----------------------------
		var raw ContentStatusResponse
		if err := json.Unmarshal(body, &raw); err != nil {
			return runtime.MCPError(
				errors.E("tag_status", errors.K.Invalid,
					"reason", "invalid JSON", "error", err),
			)
		}

		out := make([]TagStatusSummary, len(raw.Models))
		for i, m := range raw.Models {
			out[i] = TagStatusSummary{
				Model:           m.Model,
				Track:           m.Track,
				LastRun:         m.LastRun,
				PercentComplete: m.PercentCompletion,
			}
		}

		wrapped := TagStatusSummaryResponse{
			Statuses: out,
		}

		return &mcp.CallToolResult{}, wrapped, nil
	}

	// -----------------------------
	// Model‑specific mode
	// -----------------------------
	var raw ModelStatusResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return runtime.MCPError(
			errors.E("tag_status", errors.K.Invalid,
				"reason", "invalid JSON", "error", err),
		)
	}

	out := TagStatusModelDetail{
		Summary: TagStatusModelSummary{
			Model:           raw.Summary.Model,
			Track:           raw.Summary.Track,
			LastRun:         raw.Summary.LastRun,
			PercentComplete: raw.Summary.TaggingProgress,
			NumContentParts: raw.Summary.NumContentParts,
		},
		Jobs: make([]TagStatusJobDetail, len(raw.Jobs)),
	}

	for i, j := range raw.Jobs {
		out.Jobs[i] = TagStatusJobDetail{
			TimeRan: j.TimeRan,
			Params:  j.Params,
			Status:  j.JobStatus,
			Upload:  j.UploadStatus,
		}
	}

	return &mcp.CallToolResult{}, out, nil
}
