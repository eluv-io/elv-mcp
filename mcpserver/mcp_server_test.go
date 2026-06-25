package mcpserver_test

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/mcpserver"
	"github.com/qluvio/elv-mcp/tasks"
	_ "github.com/qluvio/elv-mcp/tasks/all" // triggers auto-registration
)

// ────────────────────────────────────────────────────────────────────────────────
//
//	Test 1: Auto‑registration + registry wiring
//
// ────────────────────────────────────────────────────────────────────────────────
//
// This test verifies the integration between the tasks registry and the
// MCP server wiring layer:
//
//   - Importing tasks/all triggers all task init() functions.
//   - tasks.All() returns a non‑empty list.
//   - ToolRegistry stores the tasks passed to it.
//   - RegisterAll(server) runs without error.
//
// This test does NOT verify that tools were installed into the MCP server,
// because the MCP SDK intentionally does not expose any introspection API.
func TestMCPServerInitializationRegistersAllTasks(t *testing.T) {
	cfg := &config.Config{}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "test-server",
		Version: "0.0.1",
	}, nil)

	reg := mcpserver.NewToolRegistry(cfg, tasks.All()...)
	reg.RegisterAll(server)

	got := reg.Tasks()
	if len(got) == 0 {
		t.Fatalf("expected at least one task, got none")
	}
}

// ────────────────────────────────────────────────────────────────────────────────
//
//	Test 2: RegisterAll calls Task.Register
//
// ────────────────────────────────────────────────────────────────────────────────
//
// This test verifies the core responsibility of ToolRegistry:
//
//   - RegisterAll(server) must call Register() on every task.
//
// We use a fake task to track whether Register() was invoked.
type fakeTask struct {
	called bool
}

func (f *fakeTask) Name() string        { return "fake" }
func (f *fakeTask) Description() string { return "fake desc" }
func (f *fakeTask) Register(server *mcp.Server, cfg *config.Config) {
	f.called = true
}

func TestToolRegistryCallsTaskRegister(t *testing.T) {
	cfg := &config.Config{}
	prim := &fakeTask{}
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "0.0.1"}, nil)

	reg := mcpserver.NewToolRegistry(cfg, prim)
	reg.RegisterAll(server)

	if !prim.called {
		t.Errorf("expected task.Register to be called")
	}
}
