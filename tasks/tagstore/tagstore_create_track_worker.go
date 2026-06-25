package tagstore

import (
	"bytes"
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

var LogCreate = elog.Get("/tagstore/create")

// -----------------------------------------------------------------------------
// TagStore create track worker
// -----------------------------------------------------------------------------

func TagStoreCreateTrackWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args TagStoreCreateTrackArgs,
	cfg *config.Config,
) (*mcp.CallToolResult, any, error) {

	// ---------------------- VALIDATION ----------------------
	if strings.TrimSpace(args.QID) == "" {
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Invalid, "reason", "qid is required"),
		)
	}
	if strings.TrimSpace(args.Track) == "" {
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Invalid, "reason", "track is required"),
		)
	}

	tf, ok := runtime.TenantFromContext(ctx)
	if !ok {
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Permission, "reason", "tenant not found in context"),
		)
	}

	// ---------------------- AUTH TOKEN ----------------------
	token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", args.QID)
	if err != nil {
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Permission,
				"reason", "failed to fetch editor-signed token", "error", err),
		)
	}

	// ---------------------- BUILD BODY ----------------------
	body := make(map[string]string)

	if args.Label != nil && *args.Label != "" {
		body["label"] = *args.Label
	}
	if args.Color != nil && *args.Color != "" {
		body["color"] = *args.Color
	}
	if args.Description != nil && *args.Description != "" {
		body["description"] = *args.Description
	}

	var bodyBytes []byte
	if len(body) > 0 {
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return runtime.MCPError(
				errors.E("tagstore_create_track", errors.K.Invalid,
					"reason", "failed to marshal request body", "error", err),
			)
		}
	} else {
		bodyBytes = []byte("{}")
	}

	// ---------------------- BUILD REQUEST ----------------------
	base := strings.TrimRight(cfg.TagStoreUrl, "/")
	trackEscaped := url.PathEscape(args.Track)
	urlStr := fmt.Sprintf("%s/%s/tracks/%s?authorization=%s", base, args.QID, trackEscaped, token)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Invalid,
				"reason", "failed to build tagstore create request", "error", err),
		)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	LogCreate.Debug("TagStoreCreateTrack - sending request", "URL", urlStr, "Body", string(bodyBytes))

	// ---------------------- EXECUTE REQUEST ----------------------
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Unavailable,
				"reason", "tagstore create request failed", "error", err),
		)
	}
	defer resp.Body.Close()

	// ---------------------- HANDLE RESPONSE ----------------------
	switch resp.StatusCode {
	case http.StatusCreated:
		var parsed struct {
			Message string `json:"message"`
			TrackID string `json:"track_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			return runtime.MCPError(
				errors.E("tagstore_create_track", errors.K.Invalid,
					"reason", "failed to parse tagstore create response", "error", err),
			)
		}

		return &mcp.CallToolResult{}, &TagStoreCreateTrackResult{
			QID:     args.QID,
			Track:   args.Track,
			TrackID: parsed.TrackID,
			Message: parsed.Message,
			Created: true,
		}, nil

	case http.StatusBadRequest:
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Invalid,
				"reason", "invalid input for track creation"),
		)

	case http.StatusUnauthorized:
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Permission,
				"reason", "unauthorized to create track"),
		)

	case http.StatusConflict:
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Exist,
				"reason", "track already exists"),
		)

	default:
		return runtime.MCPError(
			errors.E("tagstore_create_track", errors.K.Unavailable,
				"reason", "tagstore returned unexpected status on create", "status", resp.Status),
		)
	}
}
