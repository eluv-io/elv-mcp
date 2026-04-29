package taggers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// Types definition from OpenAPI (TrackInfo, ModelSpec, ModelsResponse)
// -----------------------------------------------------------------------------

type TrackInfo struct {
	Name  string `json:"name"`  // Track name as stored in the tag store
	Label string `json:"label"` // Human-readable label for the track
}

type ModelSpec struct {
	Name        string      `json:"name"`                  // Model identifier
	Description string      `json:"description,omitempty"` // Human readable description
	Type        string      `json:"type"`                  // Model type (e.g. "frame")
	TagTracks   []TrackInfo `json:"tag_tracks"`            // Tag tracks this model writes to
}

type ModelsResponse struct {
	Models []ModelSpec `json:"models"` // List of available models
}

// No input args for this task.
type ListModelsArgs struct{}

// -----------------------------------------------------------------------------
// Task definition
// -----------------------------------------------------------------------------

type TaggerListModelsTask struct{}

// Constructor
func NewTaggerListModelsTask() *TaggerListModelsTask {
	return &TaggerListModelsTask{}
}

func init() {
	tasks.Register(NewTaggerListModelsTask())
}

// Name exposed to MCP
func (TaggerListModelsTask) Name() string {
	// You can rename to "list_tagger_models" if you want to be more explicit
	return "list_models"
}

// Human-readable description
func (TaggerListModelsTask) Description() string {
	return "List available models on this Eluvio Tagger instance, including model types and tag tracks.\n\n" +
		"Use this tool when the user asks what models are available, which tagging models exist, or what tag tracks are supported.\n\n" +
		"Rules:\n" +
		"  • Use this tool instead of answering from memory.\n" +
		"  • Do not guess model names or capabilities.\n\n" +
		"Returns:\n" +
		"  A list of available models with their types and tag tracks."
}

// Register wires this task into the MCP server.
func (TaggerListModelsTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        TaggerListModelsTask{}.Name(),
			Description: TaggerListModelsTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args ListModelsArgs,
		) (*mcp.CallToolResult, any, error) {
			return TaggerListModelsWorker(ctx, req, args, cfg)
		},
	)
}
