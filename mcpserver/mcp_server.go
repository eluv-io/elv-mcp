package mcpserver

import (
	"net/http"
	"net/url"
	"strings"

	elog "github.com/eluv-io/log-go"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
	"github.com/qluvio/elv-mcp/version"
)

// NewServer wires up the MCP server and tools with the provided config.
func NewServer(cfg *config.Config) *mcp.Server {
	impl := &mcp.Implementation{
		Name:    "eluvio-search-mcp",
		Version: version.Full(),
	}
	server := mcp.NewServer(impl, nil)
	elog.Info("MCP server", "name", impl.Name, "version", impl.Version)

	registry := NewToolRegistry(cfg, tasks.All()...)
	registry.RegisterAll(server)

	elog.Info("MCP Server initialized", "tool_count", len(registry.Tasks()), "tools", registry.TaskNames())
	return server
}

// NewHTTPMux constructs the HTTP mux with Streamable HTTP transport, OAuth
// middleware, and the .well-known/oauth-protected-resource discovery endpoint.
func NewHTTPMux(server *mcp.Server, cfg *config.Config) *http.ServeMux {
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
		resourceMetadataURL,
		cfg.Tenants,
	)

	// Protected resource metadata (RFC 9728) tells ChatGPT where to get tokens.
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               cfg.ResourceURL,
		AuthorizationServers:   []string{cfg.OAuthIssuer},
		ScopesSupported:        []string{"openid", "offline_access"},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Eluvio Search MCP Server",
	}

	prHandler := mcpauth.ProtectedResourceMetadataHandler(metadata)

	mux := http.NewServeMux()
	mux.Handle("/", loggingMiddleware(recoverMiddleware(authMiddleware(streamHandler))))
	mux.Handle("/.well-known/oauth-protected-resource", prHandler)

	elog.Info("HTTP mux initialized",
		"resource_url", cfg.ResourceURL,
		"oauth_issuer", cfg.OAuthIssuer,
		"behind_proxy", behindProxy,
	)

	elog.Info("HTTP mux initialized",
		"resource_url", cfg.ResourceURL,
		"oauth_issuer", cfg.OAuthIssuer,
		"behind_proxy", behindProxy,
	)

	return mux
}
