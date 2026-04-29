package tasks

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// AsynchWorker retrieves the current status of a task and returns
// a snapshot suitable for MCP clients.
//
// It follows the same handler pattern as your other tasks:
//
//	(*mcp.CallToolResult, T, error)
//
// For this primitive, we rely on the SDK's default behavior when
// CallToolResult is nil: the second return value (Snapshot) will be
// marshalled as JSON content for the tool response.
func AsynchWorker(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args TaskStatusArgs,
) (*mcp.CallToolResult, *Snapshot, error) {
	if args.TaskID == "" {
		return nil, nil, fmt.Errorf("task_id is required")
	}

	snap, ok := GetSnapshot(args.TaskID)
	if !ok {
		return nil, nil, fmt.Errorf("task not found: %s", args.TaskID)
	}

	// We return nil for CallToolResult and let the SDK marshal the
	// Snapshot as JSON. This keeps the handler simple and consistent
	// with other tasks that rely on the generic handler behavior.
	return nil, &snap, nil
}
