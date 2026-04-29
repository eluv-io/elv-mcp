package tasks

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// TaskStatusArgs defines the input schema for the task_status task.
type TaskStatusArgs struct {
	TaskID string `json:"task_id"`
}

// StatusTask registers the MCP task that allows clients
// to query the status of an asynchronous task.
type StatusTask struct{}

func NewStatusTask() *StatusTask {
	return &StatusTask{}
}

func init() {
	// Register this task with the global tasks registry.
	tasks.Register(NewStatusTask())
}

// Name returns the MCP tool name exposed to the LLM.
func (StatusTask) Name() string {
	return "task_status"
}

// Description returns the human-readable description of the tool.
// This is what the LLM sees when deciding whether to call the tool.
func (StatusTask) Description() string {
	return "Retrieve the status or final result of an asynchronous task started by another Eluvio tool.\n\n" +
		"Use this tool whenever a previous tool returned a `task_id` and the user asks for progress, status, completion, or results.\n\n" +
		"Required parameter:\n" +
		"  • task_id — the identifier returned by a previous asynchronous tool.\n\n" +
		"Rules:\n" +
		"  • Only call this tool when the user refers to a task ID.\n" +
		"  • Do not guess task status.\n" +
		"  • If `task_id` is missing, state that it is required.\n\n" +
		"Returns:\n" +
		"  The current task status and, if complete, the final result."
}

// Register wires this task into the MCP server by calling AddTool.
// It binds the tool metadata to the TaskStatusHandler implementation.
func (StatusTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        StatusTask{}.Name(),
			Description: StatusTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args TaskStatusArgs,
		) (*mcp.CallToolResult, *Snapshot, error) {
			return AsynchWorker(ctx, req, args)
		},
	)
}
