package tasks

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
)

// Task is the interface implemented by all MCP Tasks.
//
// Auto‑registration pattern:
// --------------------------
// Each Task file should include a package‑level init() function:
//
//	func init() {
//	    Tasks.Register(NewMyTask())
//	}
//
// When the Task's package is imported (directly or via the aggregator
// package under Tasks/all), Go automatically runs all init() functions, causing each Task
// to self‑register into the global registry.
//
// The MCP server then calls Tasks.All() to retrieve the full list.
//
// Package Tasks defines the core abstraction used to expose capabilities
// ("tools") to the MCP server. A Task represents a single MCP-exposed
// operation, such as searching for clips or refreshing signed URLs.
var global []Task

// Register is called by each Task's init() function.
func Register(p Task) {
	global = append(global, p)
}

// All returns all registered Tasks.
func All() []Task {
	return global
}

// Each Task is responsible for:
//
//   - Declaring its tool name (as seen by the LLM)
//   - Providing a human-readable description
//   - Registering itself with an MCP server instance
//
// Tasks do not contain business logic. Instead, they delegate execution to
// worker functions located in subpackages (e.g., Tasks/fabric). This keeps
// tool metadata and registration concerns separate from functional logic.
//
// The MCP server invokes Register on each Task during startup. This allows
// new tools to be added simply by implementing the Task interface and
// including the implementation in the ToolRegistry.
//
// Tasks must register themselves at init time by calling Tasks.Register in their own
type Task interface {
	// Name returns the MCP tool name exposed to the LLM.
	// Example: "search_clips"
	Name() string

	// Description returns the human-readable description of the tool.
	// This is what the LLM sees when deciding whether to call the tool.
	Description() string

	// Register wires the Task into the MCP server by calling AddTool.
	// The Task is responsible for providing the correct worker function.
	Register(server *mcp.Server, cfg *config.Config)
}
