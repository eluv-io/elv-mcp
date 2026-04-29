package taggers

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// Primitive definition
// -----------------------------------------------------------------------------

type TaggerStartTask struct{}

// Input structure for the tag_content MCP tool.
// This mirrors the Tagger StartJobsRequest schema, plus a synchronous flag.
type TagContentArgs struct {
	QID string `json:"qid"` // required

	// Options apply to all jobs in the request. Mirrors TaggerOptions.
	Options *TaggerOptions `json:"options,omitempty"`

	// Jobs is the list of individual job specifications. Mirrors JobSpec.
	Jobs []TagJobSpec `json:"jobs"`

	// If true, the task blocks until tagging completes.
	// If false or omitted, an async task is created and task_id is returned.
	Synchronous bool `json:"synchronous,omitempty"`
}

// TaggerOptions mirrors the TaggerOptions schema from the Tagger API.
type TaggerOptions struct {
	DestinationQID  string         `json:"destination_qid,omitempty"`
	Replace         bool           `json:"replace,omitempty"`
	MaxFetchRetries int            `json:"max_fetch_retries,omitempty"`
	Scope           map[string]any `json:"scope,omitempty"`
}

// TagJobSpec mirrors the JobSpec schema from the Tagger API.
type TagJobSpec struct {
	Model       string         `json:"model"`
	ModelParams map[string]any `json:"model_params,omitempty"`
	Overrides   *TaggerOptions `json:"overrides,omitempty"`
}

// Output when synchronous=true
type TagContentSyncResult struct {
	Jobs []TagJobStatus `json:"jobs"`
}

// Output when synchronous=false
type TagContentAsyncResult struct {
	TaskID string `json:"task_id"`
}

// Normalized job status returned to MCP clients.
// Note: job_id is intentionally not exposed to avoid confusion with MCP task_id.
type TagJobStatus struct {
	Model           string   `json:"model"`
	Status          string   `json:"status"`
	TimeRunning     float64  `json:"time_running"`
	TaggingProgress string   `json:"tagging_progress"`
	MissingTags     []string `json:"missing_tags,omitempty"`
	Failed          []string `json:"failed,omitempty"`
}

// Constructor
func NewTaggerStartTask() *TaggerStartTask {
	return &TaggerStartTask{}
}

func init() {
	tasks.Register(NewTaggerStartTask())
}

// Name returns the MCP tool name exposed to the LLM.
func (TaggerStartTask) Name() string {
	return "tag_content"
}

// Description returns the human-readable description of the tool.
// Built dynamically from the current list of supported models.
func (TaggerStartTask) Description() string {
	models := GetSupportedModels()
	return BuildTagContentDescription(models)
}

// BuildTagContentDescription constructs the MCP tool description,
// including the dynamically generated list of supported models.
func BuildTagContentDescription(models []string) string {
	var b strings.Builder

	b.WriteString("Tag Fabric content using one or more models via the Eluvio Tagger API.\n\n")
	b.WriteString("Use this tool for general tagging requests or multi-model workflows.\n")
	b.WriteString("Use specialized tools such as `tag_chapters` or `tag_characters` when the user explicitly requests those workflows.\n\n")

	b.WriteString("Required parameters:\n")
	b.WriteString("  • qid — the Fabric content identifier.\n")
	b.WriteString("  • jobs — an array of tagging jobs to run.\n\n")

	// Insert dynamically generated supported model list
	b.WriteString(DescribeSupportedModels(models))
	b.WriteString("\n")

	b.WriteString("Each job in `jobs` may include:\n")
	b.WriteString("  • model — the model identifier to run.\n")
	b.WriteString("  • model_params — optional model-specific parameters (e.g., `{ \"fps\": 1.5 }`).\n")
	b.WriteString("  • overrides — optional per-job overrides of global options.\n\n")

	b.WriteString("Global `options` apply to all jobs and may include:\n")
	b.WriteString("  • destination_qid — where tags should be written.\n")
	b.WriteString("  • replace — whether to overwrite existing tags.\n")
	b.WriteString("  • max_fetch_retries — number of fetch retries before failure.\n")
	b.WriteString("  • scope — tagging scope (e.g., time range, assets list, livestream parameters).\n\n")

	b.WriteString("Execution mode:\n")
	b.WriteString("  • If `synchronous` is true, wait for all jobs to complete and return final statuses.\n")
	b.WriteString("  • If false, start the jobs and return a `task_id` for async polling via `task_status`.\n\n")

	b.WriteString("Rules:\n")
	b.WriteString("  • Use this tool for flexible or multi-model tagging requests.\n")
	b.WriteString("  • Do not use this tool when the user explicitly requests chapter or character tagging.\n")
	b.WriteString("  • Do not invent job configurations or model names.\n")
	b.WriteString("  • If required inputs are missing, state exactly which ones are required.\n")

	return b.String()
}

// Register wires this task into the MCP server by calling AddTool.
func (TaggerStartTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        TaggerStartTask{}.Name(),
			Description: TaggerStartTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args TagContentArgs,
		) (*mcp.CallToolResult, any, error) {
			return TagContentWorker(ctx, req, args, cfg)
		},
	)
}
