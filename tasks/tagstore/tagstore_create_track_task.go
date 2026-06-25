package tagstore

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// -----------------------------------------------------------------------------
// TagStore create track task (MCP tool wiring)
// -----------------------------------------------------------------------------

type TagStoreCreateTrackArgs struct {
	QID         string  `json:"qid"`
	Track       string  `json:"track"`
	Label       *string `json:"label,omitempty"`
	Color       *string `json:"color,omitempty"`
	Description *string `json:"description,omitempty"`
}

type TagStoreCreateTrackResult struct {
	QID     string `json:"qid"`
	Track   string `json:"track"`
	TrackID string `json:"track_id"`
	Message string `json:"message"`
	Created bool   `json:"created"`
}

type TagStoreCreateTrackTask struct{}

func NewTagStoreCreateTrackTask() *TagStoreCreateTrackTask {
	return &TagStoreCreateTrackTask{}
}

func init() {
	tasks.Register(NewTagStoreCreateTrackTask())
}

func (TagStoreCreateTrackTask) Name() string {
	return "tagstore_create_track"
}

func (TagStoreCreateTrackTask) Description() string {
	return "Create a TagStore track for a Fabric content object.\n\n" +
		"Use this tool when the user asks to create a new TagStore track.\n\n" +
		"Required parameters:\n" +
		"  • qid — the Fabric content identifier.\n" +
		"  • track — the unique track name.\n\n" +
		"Optional parameters:\n" +
		"  • label — human-readable name.\n" +
		"  • color — hex color code.\n" +
		"  • description — track description.\n\n" +
		"Rules:\n" +
		"  • Use only when the user explicitly requests track creation.\n" +
		"  • If required inputs are missing, state which ones are required.\n\n" +
		"Returns:\n" +
		"  A confirmation message and the created track ID."
}

func (TagStoreCreateTrackTask) Register(server *mcp.Server, cfg *config.Config) {
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        TagStoreCreateTrackTask{}.Name(),
			Description: TagStoreCreateTrackTask{}.Description(),
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args TagStoreCreateTrackArgs,
		) (*mcp.CallToolResult, any, error) {
			return TagStoreCreateTrackWorker(ctx, req, args, cfg)
		},
	)
}
