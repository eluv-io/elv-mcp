// tagger_chapters_task.go
package taggers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// Chapters tagging task (high‑level workflow)
// -----------------------------------------------------------------------------

// ChaptersTaggingArgs defines the MCP input schema for the `tag_chapters` tool.
//
// This is a higher‑level workflow on top of the Tagger API that:
//
//   - Ensures all required dependency models have completed (e.g. `speaker`)
//   - Optionally auto‑runs missing dependencies when requested
//   - Runs the `chapters` model once dependencies are satisfied
//
// The dependency graph is defined in a shared registry (see chapters worker).
type ChaptersTaggingArgs struct {
	// QID is the Fabric content identifier to tag. Required.
	QID string `json:"qid"`

	// AutoRunDependencies controls whether missing model dependencies should be
	// automatically started by this task.
	//
	// For example, for the `chapters` model, the dependency graph may include:
	//   - "speaker"
	//
	// If this flag is false or omitted and dependencies are not satisfied, the
	// task fails with a clear error explaining which models are missing and
	// that auto_run_dependencies was not specified.
	//
	// If true, the task will:
	//   - Start any missing dependency models
	//   - Wait for them to complete (in synchronous mode)
	//   - Then run the `chapters` model
	AutoRunDependencies bool `json:"auto_run_dependencies,omitempty"`

	// Synchronous controls whether the task blocks until chapters tagging
	// completes (and dependencies, if auto‑run).
	//
	//   - If true, the tool waits for all work to complete and returns a
	//     ChaptersTaggingSyncResult.
	//   - If false or omitted, the tool starts an async task and returns a
	//     ChaptersTaggingAsyncResult containing a task_id that can be polled
	//     via the MCP task API.
	Synchronous bool `json:"synchronous,omitempty"`

	// Options are global Tagger options applied to the `chapters` job (and
	// optionally to dependency jobs, depending on future evolution).
	Options *TaggerOptions `json:"options,omitempty"`
}

// ChaptersTaggingSyncResult is returned when ChaptersTaggingArgs.Synchronous is true.
//
// It mirrors the normalized Tagger job status structure used by tag_content,
// and adds AutoRanDependencies to indicate which dependency models (if any)
// were automatically started by this task.
type ChaptersTaggingSyncResult struct {
	// Jobs contains the final status of the `chapters` tagging jobs.
	Jobs []TagJobStatus `json:"jobs"`

	// AutoRanDependencies lists the dependency models that were automatically
	// started by this task (e.g. ["speaker"]). Empty if no dependencies were
	// auto‑run.
	AutoRanDependencies []string `json:"auto_ran_dependencies,omitempty"`
}

// ChaptersTaggingAsyncResult is returned when ChaptersTaggingArgs.Synchronous is false.
//
// The actual result of the async task (available via the async snapshot API)
// is a ChaptersTaggingSyncResult, which includes AutoRanDependencies.
type ChaptersTaggingAsyncResult struct {
	// TaskID is the identifier of the async task that is performing dependency
	// resolution and chapters tagging.
	TaskID string `json:"task_id"`
}

// TagChaptersTask wires the `tag_chapters` MCP tool into the server.
type TagChaptersTask struct{}

// NewTagChaptersTask constructs a new TagChaptersTask.
func NewTagChaptersTask() *TagChaptersTask {
	return &TagChaptersTask{}
}

func init() {
	tasks.Register(NewTagChaptersTask())
}

// Name returns the MCP tool name exposed to the LLM.
func (TagChaptersTask) Name() string {
	return "tag_chapters"
}

// Description returns a human‑readable description of the tool, including
// dependency semantics and the auto_run_dependencies behavior.
func (TagChaptersTask) Description() string {
	return "Run chapter tagging on Fabric content using the Eluvio Tagger API.\n\n" +
		"Use this tool when the user explicitly asks to generate chapter tags or run the `chapters` tagging workflow.\n\n" +
		"Required parameter:\n" +
		"  • qid — the Fabric content identifier.\n\n" +
		"Optional parameters:\n" +
		"  • auto_run_dependencies — if true, automatically run required models (e.g., `speaker`).\n" +
		"  • synchronous — if true, wait for completion and return final job statuses.\n" +
		"  • options — global Tagger options (destination_qid, replace, max_fetch_retries, scope).\n\n" +
		"Rules:\n" +
		"  • Use this tool only when the user specifically requests chapter tagging.\n" +
		"  • Do not use `tag_content` when the user explicitly requests chapters.\n" +
		"  • If `qid` is missing, state that it is required.\n\n" +
		"Returns:\n" +
		"  • In synchronous mode: final job statuses and auto-ran dependencies.\n" +
		"  • In async mode: a `task_id` for polling via `task_status`."
}

// Register wires this task into the MCP server by calling AddTool.
func (TagChaptersTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        TagChaptersTask{}.Name(),
			Description: TagChaptersTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args ChaptersTaggingArgs,
		) (*mcp.CallToolResult, any, error) {
			return TagChaptersWorker(ctx, req, args, cfg)
		},
	)
}
