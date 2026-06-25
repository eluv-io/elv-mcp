package fabric

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
)

// GetPublicMetaArgs defines the input arguments for the content.get_public_meta tool.
type GetPublicMetaArgs struct {
	// ContentID is the content hash, ID, or write token of the content object.
	// It will be used both as the QID for authorization and as the qhit in the struct URL.
	ContentID string `json:"content_id"`
}

// GetPublicMetaResult wraps the arbitrary JSON structure returned by the Fabric API.
type GetPublicMetaResult struct {
	// Data contains the full JSON response from /struct/meta/user/public.
	// Only "name" and "description" are guaranteed keys; all other fields are dynamic.
	Data map[string]any `json:"data"`
}

// GetPublicMetaTask implements the MCP task definition for content.get_public_meta.
type GetPublicMetaTask struct{}

// NewGetPublicMetaTask constructs a new GetPublicMetaTask.
func NewGetPublicMetaTask() *GetPublicMetaTask {
	return &GetPublicMetaTask{}
}

// Name returns the MCP tool name.
func (GetPublicMetaTask) Name() string {
	return "content.get_public_meta"
}

// Description returns a human-readable description of the tool.
func (GetPublicMetaTask) Description() string {
	var b strings.Builder
	b.WriteString("Retrieve the public user metadata for a content object from Fabric.\n\n")
	b.WriteString("Required parameters:\n")
	b.WriteString("- content_id (string): A content hash, ID, or write token.\n")
	b.WriteString("  - The hash of a finalized content object.\n")
	b.WriteString("  - A content ID (resolved to the latest version).\n")
	b.WriteString("  - The write token of a draft content object.\n\n")
	b.WriteString("Behavior:\n")
	b.WriteString("- Resolves the library ID (qlibid) for the given content_id using the Fabric metadata API.\n")
	b.WriteString("- Uses content_id as the qhit.\n")
	b.WriteString("- Issues a GET request to /qlibs/{qlibid}/q/{content_id}/struct/meta/user/public.\n")
	b.WriteString("- Returns the full JSON structure under the 'data' field.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Fails with an Invalid error if content_id is missing or empty.\n")
	b.WriteString("- Fails with a Permission error if the tenant is missing from context.\n")
	b.WriteString("- Fails with an Unavailable error if authorization or Fabric API calls fail.\n")
	b.WriteString("- Never assumes a fixed JSON structure beyond the presence of optional 'name' and 'description' keys.\n")
	return b.String()
}

// Register registers the get_public_meta tool with the MCP runtime.
func (GetPublicMetaTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        GetPublicMetaTask{}.Name(),
			Description: GetPublicMetaTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args GetPublicMetaArgs,
		) (*mcp.CallToolResult, any, error) {
			return GetPublicMetaWorker(ctx, req, args, cfg)
		},
	)
}