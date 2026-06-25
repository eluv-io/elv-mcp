package tagstore

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// TagStore list tracks task (MCP tool wiring)
// -----------------------------------------------------------------------------

// TagStoreListTracksArgs defines the MCP input schema for the
// `tagstore_list_tracks` tool.
//
// This tool calls the TagStore API to list all tracks for a given
// Fabric content object.
type TagStoreListTracksArgs struct {
    // QID is the Fabric content identifier whose tracks should be listed.
    QID string `json:"qid"`
}

// TagStoreTrack represents a single TagStore track entry.
type TagStoreTrack struct {
    ID          string `json:"id"`
    QID         string `json:"qid"`
    Name        string `json:"name"`
    Label       string `json:"label"`
    Color       string `json:"color"`
    Description string `json:"description"`
}

// TagStoreListTracksResult is returned on successful listing.
type TagStoreListTracksResult struct {
    // QID is the content identifier whose tracks were listed.
    QID string `json:"qid"`

    // Tracks is the list of tracks available for this content object.
    Tracks []TagStoreTrack `json:"tracks"`
}

// TagStoreListTracksTask wires the `tagstore_list_tracks` MCP tool.
type TagStoreListTracksTask struct{}

// NewTagStoreListTracksTask constructs a new TagStoreListTracksTask.
func NewTagStoreListTracksTask() *TagStoreListTracksTask {
    return &TagStoreListTracksTask{}
}

func init() {
    tasks.Register(NewTagStoreListTracksTask())
}

// Name returns the MCP tool name.
func (TagStoreListTracksTask) Name() string {
    return "tagstore_list_tracks"
}

// Description returns a human‑readable description of the tool.
func (TagStoreListTracksTask) Description() string {
    return "List TagStore tracks for a Fabric content object, returning full track metadata.\n\n" +
        "Use this tool when the user asks which TagStore tracks exist for a given content object.\n\n" +
        "Required parameters:\n" +
        "  • qid — the Fabric content identifier.\n\n" +
        "Rules:\n" +
        "  • Do not use this tool to delete or modify tracks.\n" +
        "  • If `qid` is missing, state that it is required.\n\n" +
        "Returns:\n" +
        "  A list of tracks (id, qid, name, label, color, description) for the specified content object."
}

// Register wires this task into the MCP server by calling AddTool.
func (TagStoreListTracksTask) Register(server *mcp.Server, cfg *config.Config) {
    mcp.AddTool(
        server,
        &mcp.Tool{
            Name:        TagStoreListTracksTask{}.Name(),
            Description: TagStoreListTracksTask{}.Description(),
        },
        func(
            ctx context.Context,
            req *mcp.CallToolRequest,
            args TagStoreListTracksArgs,
        ) (*mcp.CallToolResult, any, error) {
            return TagStoreListTracksWorker(ctx, req, args, cfg)
        },
    )
}