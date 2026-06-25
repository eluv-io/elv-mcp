package fabric

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/eluv-io/errors-go"
	elog "github.com/eluv-io/log-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/auth"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/runtime"
)

// GetPublicMetaWorker performs the HTTP call to Fabric to retrieve public user metadata.
func GetPublicMetaWorker(
    ctx context.Context,
    req *mcp.CallToolRequest,
    args GetPublicMetaArgs,
    cfg *config.Config,
) (*mcp.CallToolResult, any, error) {
    log := elog.Get("/fabric/get_public_meta")

    contentID := strings.TrimSpace(args.ContentID)
    if contentID == "" {
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Invalid,
            "reason", "missing required field 'content_id'"))
    }

    tf, ok := runtime.TenantFromContext(ctx)
    if !ok {
        return runtime.MCPError(			
			errors.E("content.get_public_meta", errors.K.Permission,
            "reason", "no tenant configuration found for this user"))
    }
	
    token, err := auth.Auth.FetchEditorSigned(cfg, tf, "", contentID)
    if err != nil {
        log.Error("failed to fetch editor-signed token", "error", err)
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Unavailable,
            "reason", "failed to fetch editor-signed token", "error", err))
    }

    url, err := BuildPublicMetaURL(cfg, contentID, token)
    if err != nil {
        log.Error("failed to build public metadata URL", "error", err)
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Unavailable,
            "reason", "failed to build public metadata URL", "error", err))
    }

    httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        log.Error("failed to create HTTP request", "error", err)
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Unavailable,
            "reason", "failed to create HTTP request", "error", err))
    }

    httpReq.Header.Set("Authorization", "Bearer "+token)
    httpReq.Header.Set("Accept", "application/json")

    resp, err := http.DefaultClient.Do(httpReq)
    if err != nil {
        log.Error("HTTP request failed", "error", err)
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Unavailable,
            "reason", "HTTP request failed", "error", err))
    }
    defer resp.Body.Close()

    switch resp.StatusCode {
    case http.StatusOK:
        var data map[string]any
        if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
            log.Error("failed to decode response JSON", "error", err)
            return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Unavailable,
                "reason", "failed to decode response JSON", "error", err))
        }
        res := &mcp.CallToolResult{
            IsError: false,
        }
        return res, GetPublicMetaResult{Data: data}, nil

    case http.StatusBadRequest:
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Invalid,
            "reason", "bad request to Fabric struct API", "status", resp.StatusCode))

    case http.StatusUnauthorized, http.StatusForbidden:
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Permission,
            "reason", "unauthorized to access Fabric struct API", "status", resp.StatusCode))

    case http.StatusNotFound, http.StatusConflict:
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Exist,
            "reason", "content object or struct path not found", "status", resp.StatusCode))

    default:
        return runtime.MCPError(errors.E("content.get_public_meta", errors.K.Unavailable,
            "reason", "unexpected Fabric response status", "status", resp.StatusCode))
    }
}
