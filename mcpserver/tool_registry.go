package mcpserver

import (
	elog "github.com/eluv-io/log-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

/*
Package mcpserver provides the ToolRegistry, a lightweight orchestrator
responsible for wiring all MCP tasks into a running MCP server.

The ToolRegistry does not contain any business logic. Instead, it coordinates
the registration of independent tasks, each of which implements the
`tasks.Task` interface and knows how to register its own MCP tool.

This design provides several benefits:

  • Clear separation of concerns:
      - Tasks define tool metadata and worker wiring
      - Workers implement business logic
      - The registry simply orchestrates registration

  • Extensibility:
      Adding a new tool requires only implementing a new Task and passing
      it to the registry. No central switch statements or manual wiring.

  • Testability:
      Tasks can be tested independently, and the registry can be tested
      by verifying that each task’s Register method is invoked.

Usage:

  cfg := tools.LoadConfig()
  registry := mcpserver.NewToolRegistry(cfg,
      fabric.SearchClipsTask{},
      fabric.RefreshURLTask{},
  )

  server := mcp.NewServer()
  registry.RegisterAll(server)

After registration, the MCP server exposes all tools defined by the provided
tasks, each with its own name, description, and worker logic.
*/

// ToolRegistry owns the list of tasks and the shared config.
// It is responsible only for orchestrating registration.
type ToolRegistry struct {
	cfg   *config.Config
	tasks []tasks.Task
}

// NewToolRegistry constructs a registry with the provided tasks.
func NewToolRegistry(cfg *config.Config, ps ...tasks.Task) *ToolRegistry {
	return &ToolRegistry{
		cfg:   cfg,
		tasks: ps,
	}
}

// RegisterAll registers all tasks with the MCP server.
// Each task handles its own metadata and worker wiring.
func (r *ToolRegistry) RegisterAll(server *mcp.Server) {
	for _, p := range r.tasks {
		elog.Info("Registering", "Task", p.Name())
		p.Register(server, r.cfg)
	}
}

func (r *ToolRegistry) Tasks() []tasks.Task {
	return r.tasks
}

func (r *ToolRegistry) TaskNames() []string {
	names := make([]string, len(r.tasks))
	for i, task := range r.tasks {
		names[i] = task.Name()
	}
	return names
}

/*
Testing the ToolRegistry is non-trivial because it does not expose any
observable state about registered tools. The only way to verify that a tool
was registered is to ensure that the task's Register() method was invoked.

To test this, we create a fake task implementation that sets a flag when
Register() is called. We then verify that the flag is set after calling
RegisterAll().

This approach allows us to confirm that the registry correctly invokes each
task's Register() method without needing to inspect the internal state of
the MCP server or the registry itself.

Note: We do NOT and CANNOT test the server’s internal tool registry. We only verify that Register() was called on each task and that AddTool() did not panic.
*/
