package tagstore

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/eluv-io/errors-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	elog "github.com/eluv-io/log-go"
	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

var Log = elog.Get("/tagstore")

// -----------------------------------------------------------------------------
// TagStore delete track worker
// -----------------------------------------------------------------------------

// TagStoreDeleteTrackWorker performs a synchronous delete of a single track
// (and all associated batches/tags) via the TagStore API.
//
// It follows the same MCP error contract as other workers:
//   - On error: returns non‑nil *mcp.CallToolResult with IsError=true, nil payload, and error.
//   - On success: returns non‑error result and a TagStoreDeleteTrackResult payload.
func TagStoreDeleteTrackWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args TagStoreDeleteTrackArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// ---------------------- VALIDATION ----------------------
	if strings.TrimSpace(args.QID) == "" {
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Invalid, "reason", "qid is required"),
		)
	}
	if strings.TrimSpace(args.Track) == "" {
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Invalid, "reason", "track is required"),
		)
	}

	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Permission, "reason", "tenant not found in context"),
		)
	}

	Log.Debug("TagStoreDeleteTrack - starting", "QID", args.QID, "Track", args.Track)

	// ---------------------- AUTH TOKEN ----------------------
	token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", args.QID)
	if err != nil {
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Permission,
				"reason", "failed to fetch editor-signed token", "error", err),
		)
	}

	// ---------------------- BUILD REQUEST ----------------------
	base := strings.TrimRight(cfg.TagStoreUrl, "/")
	trackEscaped := url.PathEscape(args.Track)
	urlStr := fmt.Sprintf("%s/%s/tracks/%s?authorization=%s", base, args.QID, trackEscaped, token)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, urlStr, nil)
	if err != nil {
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Invalid,
				"reason", "failed to build tagstore delete request", "error", err),
		)
	}
	httpReq.Header.Set("Accept", "application/json")

	Log.Debug("TagStoreDeleteTrack - sending request", "URL", urlStr)

	// ---------------------- EXECUTE REQUEST ----------------------
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Unavailable,
				"reason", "tagstore delete request failed", "error", err),
		)
	}
	defer resp.Body.Close()

	// ---------------------- HANDLE RESPONSE ----------------------
	switch resp.StatusCode {
	case http.StatusNoContent:
		// Success
		Log.Debug("TagStoreDeleteTrack - success", "QID", args.QID, "Track", args.Track)
		return &mcp.CallToolResult{}, &TagStoreDeleteTrackResult{
			QID:     args.QID,
			Track:   args.Track,
			Deleted: true,
		}, nil

	case http.StatusUnauthorized:
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Permission,
				"reason", "unauthorized to delete track"),
		)

	case http.StatusNotFound:
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.NotFound,
				"reason", "track not found"),
		)

	default:
		return runtime.MCPError(
			errors.E("tagstore_delete_track", errors.K.Unavailable,
				"reason", "tagstore returned unexpected status on delete", "status", resp.Status),
		)
	}
}
