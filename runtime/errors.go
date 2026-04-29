package runtime

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/tasks"
)

// ToolError returns a failed MCP tool result (IsError=true) and logs it.
func ToolError(userMessage string, err error) (*mcp.CallToolResult, *tasks.ClipResponse, error) {
	if err != nil {
		Log.Error("tool error", "message", userMessage, "error", err)
	} else {
		Log.Error("tool error", "message", userMessage)
	}

	text := userMessage
	if err != nil {
		// For security reasons, we don't want to expose internal error details to the user, but we include a generic message and log the details on the server.
		text = fmt.Sprintf("%s: %v", userMessage, "Check server logs for details")
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		IsError: true,
	}, nil, nil
}

// MCPError wraps an error into a correct MCP tool failure response.
// Always returns:
//   - CallToolResult{IsError: true}
//   - nil result payload
//   - the provided error
func MCPError(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{IsError: true}, nil, err
}

func MCPErrorWithResult(err error, result any) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{IsError: true}, result, err
}
