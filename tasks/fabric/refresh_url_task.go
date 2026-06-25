package fabric

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

/*
RefreshURLTask implements the Task interface for the "refresh_clips"
MCP tool.

This task is responsible for:

  - Exposing the tool name used by the LLM ("refresh_clips")
  - Providing the human-readable description of the tool
  - Registering the tool with the MCP server
  - Wiring the tool to the underlying business logic (RefreshURLWorker)

The task does NOT contain business logic itself. Instead, it delegates
execution to fabric.RefreshURLWorker, keeping registration concerns separate
from functional concerns.

This design mirrors an object-oriented command pattern: each task is a
self-contained unit that knows how to describe and register itself.
*/
type RefreshURLTask struct{}

type RefreshClipsArgs struct {
	Contents []tasks.ClipItem `json:"contents"`
}

func NewRefreshURLTask() *RefreshURLTask {
	return &RefreshURLTask{}
}

func init() {
	tasks.Register(NewRefreshURLTask())
}

// Name returns the MCP tool name exposed to the LLM.
func (RefreshURLTask) Name() string {
	return "refresh_clips"
}

// Description returns the human-readable description of the tool.
// This is what the LLM sees when deciding whether to call the tool.
func (RefreshURLTask) Description() string {
	return "Refresh authentication tokens in existing Eluvio clip and thumbnail URLs.\n\n" +
		"Use this tool when clip URLs or thumbnail URLs were previously returned and need refreshed authentication before reuse or display.\n\n" +
		"Required input:\n" +
		"  • A list of clip or image URLs previously returned by another tool.\n\n" +
		"Rules:\n" +
		"  • Only use this tool when clip or image URLs already exist in the conversation.\n" +
		"  • Do not invent or guess URLs.\n" +
		"  • Do not use this tool to search for clips.\n" +
		"  • If required URLs are missing, state what is required.\n\n" +
		"Returns:\n" +
		"  The same clip and image URLs with refreshed authentication tokens."
}

// Register wires this task into the MCP server by calling AddTool.
// It binds the tool metadata to the RefreshURLWorker implementation.
func (RefreshURLTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        RefreshURLTask{}.Name(),
			Description: RefreshURLTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args RefreshClipsArgs,
		) (*mcp.CallToolResult, *tasks.ClipResponse, error) {
			return RefreshURLWorker(ctx, req, args, cfg)
		},
	)
}
