package mcpserver

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
)

/*
fakePrimitive is used because the MCP Go SDK does NOT expose any public API
to introspect which tools were registered on an mcp.Server.

- No server.ListTools()
- No server.Tools field
- No capabilities reflecting AddTool()
- No worker wrapping or interception

Therefore, the ONLY reliable unit-test strategy is to verify:

 1. Register() was called
 2. mcp.AddTool() did not panic

We do NOT and CANNOT test the server’s internal tool registry.
*/
type fakePrimitive struct {
	name        string
	description string
	registered  bool
}

func (f *fakePrimitive) Name() string        { return f.name }
func (f *fakePrimitive) Description() string { return f.description }

func (f *fakePrimitive) Register(server *mcp.Server, cfg *config.Config) {
	f.registered = true

	// We cannot detect AddTool invocation; we only ensure it does not panic.
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        f.name,
			Description: f.description,
		},
		func(ctx context.Context, req *mcp.CallToolRequest, args struct{}) (*mcp.CallToolResult, struct{}, error) {
			return &mcp.CallToolResult{}, struct{}{}, nil
		},
	)
}

func TestToolRegistry_RegisterAll(t *testing.T) {
	cfg := &config.Config{}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "0.0.1",
	}, nil)

	p1 := &fakePrimitive{
		name:        "search_clips",
		description: "Searches the Eluvio Search API and returns video clips.",
	}
	p2 := &fakePrimitive{
		name:        "refresh_clips",
		description: "Refreshes auth tokens in existing clip and image URLs.",
	}

	registry := NewToolRegistry(cfg, p1, p2)

	// Ensure no panic occurs during registration
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("RegisterAll panicked: %v", r)
		}
	}()

	registry.RegisterAll(server)

	// Ensure Register() was invoked
	if !p1.registered {
		t.Fatalf("primitive %s was not registered", p1.name)
	}
	if !p2.registered {
		t.Fatalf("primitive %s was not registered", p2.name)
	}
}
