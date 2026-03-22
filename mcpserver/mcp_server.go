package mcpserver

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/qluvio/elv-mcp/types"
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

// NewHTTPMux constructs the HTTP mux with Streamable HTTP transport, OAuth
// middleware, and the .well-known/oauth-protected-resource discovery endpoint.
func NewHTTPMux(server *mcp.Server, cfg *types.Config) *http.ServeMux {
	// Disable localhost DNS rebinding protection when serving behind a reverse
	// proxy (e.g. ngrok) — the socket is loopback but the Host header is the
	// public hostname, which the SDK would otherwise reject.
	behindProxy := false
	if u, err := url.Parse(cfg.ResourceURL); err == nil {
		host := strings.ToLower(u.Hostname())
		behindProxy = host != "localhost" && host != "127.0.0.1" && host != "::1"
	}

	streamHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return server },
		&mcp.StreamableHTTPOptions{
			DisableLocalhostProtection: behindProxy,
		},
	)

	resourceMetadataURL := cfg.ResourceURL + "/.well-known/oauth-protected-resource"

	// Selective auth: initialize and notifications/initialized pass through,
	// everything else (tools/list, tools/call) requires OAuth bearer token.
	authMiddleware := selectiveAuthMiddleware(
		NewTokenVerifier(cfg),
		&auth.RequireBearerTokenOptions{
			ResourceMetadataURL: resourceMetadataURL,
		},
	)

	// Protected resource metadata (RFC 9728) tells ChatGPT where to get tokens.
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               cfg.ResourceURL,
		AuthorizationServers:   []string{cfg.OAuthIssuer},
		ScopesSupported:        []string{"openid", "offline_access"},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Eluvio Search MCP Server",
	}

	prHandler := auth.ProtectedResourceMetadataHandler(metadata)

	mux := http.NewServeMux()
	mux.Handle("/", loggingMiddleware(recoverMiddleware(authMiddleware(streamHandler))))
	mux.Handle("/.well-known/oauth-protected-resource", prHandler)

	return mux
}
