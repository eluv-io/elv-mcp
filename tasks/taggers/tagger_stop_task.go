package taggers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// Primitive definition
// -----------------------------------------------------------------------------

type TaggerStopTask struct{}

// Input structure for the stop_tagging MCP tool.
type TaggerStopArgs struct {
	QID   string `json:"qid"`             // required
	Model string `json:"model,omitempty"` // optional: if empty → stop all jobs
}

// Output structure (always synchronous)
type TaggerStopResult struct {
	Jobs []TagStopStatus `json:"jobs"`
}

// Normalized stop status returned to MCP clients.
type TagStopStatus struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

// Constructor
func NewTaggerStopTask() *TaggerStopTask {
	return &TaggerStopTask{}
}

func init() {
	tasks.Register(NewTaggerStopTask())
}

// Name returns the MCP tool name exposed to the LLM.
func (TaggerStopTask) Name() string {
	return "stop_tagging"
}

// Description returns the human-readable description of the tool.
func (TaggerStopTask) Description() string {
	return "Stop tagging jobs for a Fabric content object.\n\n" +
		"Use this tool only when the user explicitly asks to stop or cancel tagging.\n\n" +
		"Required parameter:\n" +
		"  • qid — the Fabric content identifier.\n\n" +
		"Optional parameter:\n" +
		"  • model — if provided, stop only jobs for that model; otherwise stop all jobs.\n\n" +
		"Rules:\n" +
		"  • This is an interruptive operation; use only when clearly requested.\n" +
		"  • If `qid` is missing, state that it is required.\n\n" +
		"Returns:\n" +
		"  A list of stopped jobs with job IDs and status messages."
}

// Register wires this task into the MCP server.
func (TaggerStopTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        TaggerStopTask{}.Name(),
			Description: TaggerStopTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args TaggerStopArgs,
		) (*mcp.CallToolResult, any, error) {
			return TaggerStopWorker(ctx, req, args, cfg)
		},
	)
}
