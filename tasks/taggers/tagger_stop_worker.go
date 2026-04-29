package taggers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/eluv-io/errors-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

// -----------------------------------------------------------------------------
// Internal request/response structures for Tagger API
// -----------------------------------------------------------------------------

type StopTaggingResponse struct {
	Jobs    []StopStatus `json:"jobs"`
	Message string       `json:"message"`
}

type StopStatus struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

// -----------------------------------------------------------------------------
// Public handler entrypoint (always synchronous)
// -----------------------------------------------------------------------------

func TaggerStopWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args TaggerStopArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// Validate input
	if strings.TrimSpace(args.QID) == "" {
		return runtime.MCPError(
			errors.E("tag_content", errors.K.Invalid, "reason", "qid is required"),
		)
	}

	// Resolve tenant
	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("stop_tagging", errors.K.Permission, "reason", "tenant not found in context"),
		)
	}

	// Execute stop operation synchronously
	jobs, err := runTaggerStop(ctx, args, cfg, tf)
	if err != nil {
		return runtime.MCPErrorWithResult(err, &TaggerStopResult{Jobs: jobs})
	}

	return &mcp.CallToolResult{}, &TaggerStopResult{Jobs: jobs}, nil
}

// -----------------------------------------------------------------------------
// Synchronous stop execution
// -----------------------------------------------------------------------------

func runTaggerStop(
	ctx context.Context,
	args TaggerStopArgs,
	cfg *config.Config,
	tf *config.TenantFabric,
) ([]TagStopStatus, error) {

	token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", args.QID)
	if err != nil {
		return nil, errors.E("stop_tagging", errors.K.Permission,
			"reason", "failed to fetch editor-signed token", "error", err)
	}

	// Build URL
	base := strings.TrimRight(cfg.AITaggerUrl, "/")
	var url string
	if args.Model == "" {
		url = fmt.Sprintf("%s/%s/stop?authorization=%s", base, args.QID, token)
	} else {
		url = fmt.Sprintf("%s/%s/stop/%s?authorization=%s", base, args.QID, NormalizeModelName(args.Model), token)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, errors.E("stop_tagging", errors.K.Invalid,
			"reason", "failed to build tagger stop request", "error", err)
	}
	// httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Accept", "application/json")

	Log.Debug("Stop tagger", "RequestUrl", url)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, errors.E("stop_tagging", errors.K.Unavailable,
			"reason", "tagger stop request failed", "error", err)
	}

	Log.Debug("Stop tagger", "Response", resp)

	defer resp.Body.Close()

	var parsed StopTaggingResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, errors.E("stop_tagging", errors.K.Invalid,
			"reason", "failed to parse tagger stop response", "error", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			out := make([]TagStopStatus, 1)
			out[0] = TagStopStatus{
				JobID:   "",
				Message: parsed.Message,
			}
			return out, errors.E("stop_tagging", errors.K.NotFound,
				"reason", parsed.Message, "status", resp.Status)
		} else {
			return nil, errors.E("stop_tagging", errors.K.NotExist,
				"reason", "tagger returned non-200 on stop", "status", resp.Status)
		}
	}

	// Normalize output
	out := make([]TagStopStatus, len(parsed.Jobs))
	for i, j := range parsed.Jobs {
		out[i] = TagStopStatus{
			JobID:   j.JobID,
			Message: j.Message,
		}
	}

	return out, nil
}
