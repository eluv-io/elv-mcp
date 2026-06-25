package taggers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// OpenAPI‑mirroring types (raw decode targets)
// -----------------------------------------------------------------------------

type ContentStatusResponse struct {
	Models []ContentModelStatusSummary `json:"models"`
}

type ContentModelStatusSummary struct {
	Model             string  `json:"model"`
	Track             string  `json:"track"`
	LastRun           string  `json:"last_run"`
	PercentCompletion float64 `json:"percent_completion"`
}

type ModelStatusResponse struct {
	Summary ModelStatusSummary `json:"summary"`
	Jobs    []JobDetail        `json:"jobs"`
}

type ModelStatusSummary struct {
	Model           string  `json:"model"`
	Track           string  `json:"track"`
	LastRun         string  `json:"last_run"`
	TaggingProgress float64 `json:"tagging_progress"`
	NumContentParts int     `json:"num_content_parts"`
}

type JobDetail struct {
	TimeRan      string         `json:"time_ran"`
	SourceQID    string         `json:"source_qid"`
	Params       map[string]any `json:"params"`
	JobStatus    map[string]any `json:"job_status"`
	UploadStatus map[string]any `json:"upload_status"`
}

// -----------------------------------------------------------------------------
// Normalized MCP-facing types
// -----------------------------------------------------------------------------

type TagStatusSummary struct {
	Model           string  `json:"model"`
	Track           string  `json:"track"`
	LastRun         string  `json:"last_run"`
	PercentComplete float64 `json:"percent_complete"`
}

type TagStatusModelDetail struct {
	Summary TagStatusModelSummary `json:"summary"`
	Jobs    []TagStatusJobDetail  `json:"jobs"`
}

type TagStatusModelSummary struct {
	Model           string  `json:"model"`
	Track           string  `json:"track"`
	LastRun         string  `json:"last_run"`
	PercentComplete float64 `json:"percent_complete"`
	NumContentParts int     `json:"num_content_parts"`
}

type TagStatusJobDetail struct {
	TimeRan string         `json:"time_ran"`
	Params  map[string]any `json:"params"`
	Status  map[string]any `json:"status"`
	Upload  map[string]any `json:"upload"`
}

type TagStatusSummaryResponse struct {
	Statuses []TagStatusSummary `json:"statuses"`
}

// -----------------------------------------------------------------------------
// Task definition
// -----------------------------------------------------------------------------

type TaggerTagStatusArgs struct {
	QID   string `json:"qid"`
	Model string `json:"model,omitempty"`
}

type TaggerTagStatusTask struct{}

func NewTaggerTagStatusTask() *TaggerTagStatusTask {
	return &TaggerTagStatusTask{}
}

func init() {
	tasks.Register(NewTaggerTagStatusTask())
}

func (TaggerTagStatusTask) Name() string {
	return "tag_status"
}

func (TaggerTagStatusTask) Description() string {
	return "Retrieve tagging status for a Fabric content object, optionally filtered by model.\n\n" +
		"Use this tool when the user asks for tagging progress, current status, or completion state.\n\n" +
		"Required parameter:\n" +
		"  • qid — the Fabric content identifier.\n\n" +
		"Optional parameter:\n" +
		"  • model — return status only for that model.\n\n" +
		"Rules:\n" +
		"  • Use this tool instead of guessing whether tagging has completed.\n" +
		"  • If `qid` is missing, state that it is required.\n\n" +
		"Returns:\n" +
		"  The current tagging status, optionally filtered by model."
}

func (TaggerTagStatusTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        TaggerTagStatusTask{}.Name(),
			Description: TaggerTagStatusTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args TaggerTagStatusArgs,
		) (*mcp.CallToolResult, any, error) {
			return TaggerTagStatusWorker(ctx, req, args, cfg)
		},
	)
}
