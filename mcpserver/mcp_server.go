package mcpserver

import (
	"context"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp-experiment/types"
)

// NewServer wires up the MCP server and tools with the provided config.
func NewServer(cfg *types.Config) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "eluvio-search-mcp",
		Version: "0.9.0",
	}, nil)

	mcp.AddTool[types.SearchClipsArgs](server, &mcp.Tool{
		Name:        "search_clips",
		Description: "Searches the Eluvio Search API and returns video clips.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args types.SearchClipsArgs) (*mcp.CallToolResult, any, error) {
		return SearchClips(ctx, req, args, cfg)
	})

	mcp.AddTool[types.RefreshClipsArgs](server, &mcp.Tool{
		Name:        "refresh_clips",
		Description: "Refreshes auth tokens in existing clip and image URLs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args types.RefreshClipsArgs) (*mcp.CallToolResult, any, error) {
		return RefreshToken(ctx, req, args, cfg)
	})

	return server
}

// NewHTTPMux constructs the HTTP mux and SSE handler.
func NewHTTPMux(server *mcp.Server) *http.ServeMux {
	sseHandler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server { return server }, nil)

	mux := http.NewServeMux()
	// Wrap with recovery & simple logging so panics / handler issues don't kill the process
	mux.Handle("/mcp", loggingMiddleware(recoverMiddleware(sseHandler)))

	return mux
}
