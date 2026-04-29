package tagstore

import (
	"context"
	"encoding/json"
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

var listLog = elog.Get("/tagstore/list_tracks")

// -----------------------------------------------------------------------------
// TagStore list tracks worker
// -----------------------------------------------------------------------------

// TagStoreListTracksWorker performs a synchronous list of all tracks for a
// given content object via the TagStore API.
//
// It follows the same MCP error contract as other workers:
//   - On error: returns non‑nil *mcp.CallToolResult with IsError=true, nil payload, and error.
//   - On success: returns non‑error result and a TagStoreListTracksResult payload.
func TagStoreListTracksWorker(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args TagStoreListTracksArgs,
    cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

    // ---------------------- VALIDATION ----------------------
    if strings.TrimSpace(args.QID) == "" {
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.Invalid, "reason", "qid is required"),
        )
    }

    tf, ok := runtime.TenantFromContext(ctx)
    if !ok {
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.Permission, "reason", "tenant not found in context"),
        )
    }

    listLog.Debug("TagStoreListTracks - starting", "QID", args.QID)

    // ---------------------- AUTH TOKEN ----------------------
    token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", args.QID)
    if err != nil {
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.Permission,
                "reason", "failed to fetch editor-signed token", "error", err),
        )
    }

    // ---------------------- BUILD REQUEST ----------------------
    base := strings.TrimRight(cfg.TagStoreUrl, "/")
    qidEscaped := url.PathEscape(args.QID)
    urlStr := fmt.Sprintf("%s/%s/tracks?authorization=%s", base, qidEscaped, token)

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
    if err != nil {
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.Invalid,
                "reason", "failed to build tagstore list tracks request", "error", err),
        )
    }
    httpReq.Header.Set("Accept", "application/json")

    listLog.Debug("TagStoreListTracks - sending request", "URL", urlStr)

    // ---------------------- EXECUTE REQUEST ----------------------
    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.Unavailable,
                "reason", "tagstore list tracks request failed", "error", err),
        )
    }
    defer resp.Body.Close()

    // ---------------------- HANDLE RESPONSE ----------------------
    switch resp.StatusCode {
    case http.StatusOK:
        var apiResp struct {
            Tracks []TagStoreTrack `json:"tracks"`
        }
        if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
            return runtime.MCPError(
                errors.E("tagstore_list_tracks", errors.K.Invalid,
                    "reason", "failed to decode tagstore list tracks response", "error", err),
            )
        }

        listLog.Debug("TagStoreListTracks - success", "QID", args.QID, "TrackCount", len(apiResp.Tracks))

        return &mcp.CallToolResult{}, &TagStoreListTracksResult{
            QID:    args.QID,
            Tracks: apiResp.Tracks,
        }, nil

    case http.StatusUnauthorized:
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.Permission,
                "reason", "unauthorized to list tracks"),
        )

    case http.StatusNotFound:
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.NotFound,
                "reason", "content not found or no tracks available"),
        )

    default:
        return runtime.MCPError(
            errors.E("tagstore_list_tracks", errors.K.Unavailable,
                "reason", "tagstore returned unexpected status on list tracks", "status", resp.Status),
        )
    }
}