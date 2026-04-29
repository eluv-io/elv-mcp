package tagstore

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// TagStore delete track task (MCP tool wiring)
// -----------------------------------------------------------------------------

// TagStoreDeleteTrackArgs defines the MCP input schema for the
// `tagstore_delete_track` tool.
//
// This tool calls the TagStore API to permanently delete a single track
// (and all associated batches/tags) for a given content object.
type TagStoreDeleteTrackArgs struct {
	// QID is the Fabric content identifier whose track should be deleted.
	QID string `json:"qid"`

	// Track is the name of the track to delete.
	Track string `json:"track"`
}

// TagStoreDeleteTrackResult is returned on successful deletion.
type TagStoreDeleteTrackResult struct {
	// QID is the content identifier whose track was targeted.
	QID string `json:"qid"`

	// Track is the track name that was requested for deletion.
	Track string `json:"track"`

	// Deleted indicates whether the track was deleted successfully.
	Deleted bool `json:"deleted"`
}

// TagStoreDeleteTrackTask wires the `tagstore_delete_track` MCP tool.
type TagStoreDeleteTrackTask struct{}

// NewTagStoreDeleteTrackTask constructs a new TagStoreDeleteTrackTask.
func NewTagStoreDeleteTrackTask() *TagStoreDeleteTrackTask {
	return &TagStoreDeleteTrackTask{}
}

func init() {
	tasks.Register(NewTagStoreDeleteTrackTask())
}

// Name returns the MCP tool name.
func (TagStoreDeleteTrackTask) Name() string {
	return "tagstore_delete_track"
}

// Description returns a human‑readable description of the tool.
func (TagStoreDeleteTrackTask) Description() string {
	return "Delete a TagStore track for a Fabric content object, including all associated batches and tags.\n\n" +
		"Use this tool only when the user explicitly asks to delete a TagStore track.\n\n" +
		"Required parameters:\n" +
		"  • qid — the Fabric content identifier.\n" +
		"  • track — the track name to delete.\n\n" +
		"Rules:\n" +
		"  • This is a destructive operation; use only when clearly requested.\n" +
		"  • Do not use this tool for inspection or status checks.\n" +
		"  • If required inputs are missing, state which ones are required.\n\n" +
		"Returns:\n" +
		"  A confirmation payload or a structured error on failure."
}

// Register wires this task into the MCP server by calling AddTool.
func (TagStoreDeleteTrackTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        TagStoreDeleteTrackTask{}.Name(),
			Description: TagStoreDeleteTrackTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args TagStoreDeleteTrackArgs,
		) (*mcp.CallToolResult, any, error) {
			return TagStoreDeleteTrackWorker(ctx, req, args, cfg)
		},
	)
}
