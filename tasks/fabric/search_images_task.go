package fabric

import (
	"context"

	elog "github.com/eluv-io/log-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qluvio/elv-mcp/config"
	"github.com/qluvio/elv-mcp/tasks"
)

// SearchImagesTask implements the Task interface for the "search_images"
// MCP tool.
//
// This task is responsible for:
//
//   - Exposing the tool name used by the LLM ("search_images")
//   - Providing the human-readable description of the tool
//   - Registering the tool with the MCP server
//   - Wiring the tool to the underlying business logic (SearchImagesWorker)
//
// The task does NOT contain business logic itself. Instead, it delegates
// execution to fabric.SearchImagesWorker, keeping registration concerns
// separate from functional concerns.
type SearchImagesTask struct{}

// SearchImagesArgs is the input structure for the search_images MCP tool.
//
// Semantics:
//   - If Query is non-empty → text-based image search.
//   - Else if ImagePath is non-empty → image-upload-based search.
//   - CollectionID is optional; if empty, the worker will fall back to the
//     per-tenant default SearchCollectionID from config.TenantFabric.
type SearchImagesArgs struct {
	CollectionID string `json:"collection_id,omitempty"`
	Query        string `json:"query,omitempty"`
	ImagePath    string `json:"image_path,omitempty"`
	Image        string `json:"image,omitempty"` // This is needed to capture the MCP 'image' parameter
}

func NewSearchImagesTask() *SearchImagesTask {
	return &SearchImagesTask{}
}

func init() {
	tasks.Register(NewSearchImagesTask())
}

// Name returns the MCP tool name exposed to the LLM.
func (SearchImagesTask) Name() string {
	return "search_images"
}

// Description returns the human-readable description of the tool.
func (SearchImagesTask) Description() string {
	return "Search Eluvio for images matching a user-provided text query or an uploaded reference image.\n\n" +
		"Use this tool whenever the user asks to search for images, find images, look up images, or retrieve images " +
		"matching a phrase, keyword, or example image.\n\n" +
		"Parameters:\n" +
		"  • query — a plain text search string from the user's request (example: \"chalkboard\").\n" +
		"  • image — an uploaded file used as the visual reference for image-based search. LibreChat will attach the " +
		"uploaded file and the MCP SDK will resolve it to a temporary local file path, which is provided to the worker " +
		"as `image_path`.\n" +
		"  • collection_id — optional; if omitted, a per-tenant default collection will be used when configured.\n\n" +
		"Rules:\n" +
		"  • Provide either `query` or `image` (but not both).\n" +
		"  • If both are empty, do not call this tool.\n" +
		"  • Do not answer from memory when image search is requested.\n" +
		"  • Do not output command-line syntax.\n\n" +
		"Example call:\n" +
		"  {\"query\": \"chalkboard\"}"
}

// Register wires this task into the MCP server by calling AddTool.
// It binds the tool metadata to the SearchImagesWorker implementation.
func (SearchImagesTask) Register(server *mcp.Server, cfg *config.Config) {

	// IMPORTANT:
	// LibreChat cannot infer that `image_path` should accept an uploaded file,
	// because the auto-generated schema from the Go struct treats it as a plain string.
	//
	// To enable file uploads, we must explicitly declare a schema property with:
	//   { "type": "file", "source": "uploaded" }
	//
	// This tells LibreChat to attach the uploaded file and instructs the MCP SDK
	// to receive the file bytes and write them to a temporary file on the MCP server.
	// The SDK then populates args.ImagePath with the resolved local path.
	//
	// Without this InputSchema block, file-based image search will NEVER work.
	mcp.AddTool(
		server,
		&mcp.Tool{
			Name:        SearchImagesTask{}.Name(),
			Description: SearchImagesTask{}.Description(),
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type": "string",
					},
					"image": map[string]any{
						"type":   "string",
						"format": "file",
						"contentMediaType": "application/octet-stream",
					},
					"collection_id": map[string]any{
						"type": "string",
					},
				},
			},
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			args SearchImagesArgs,
		) (*mcp.CallToolResult, any, error) {
			// --- IMPORTANT FIX ---
			// LibreChat sends the uploaded file as the "image" parameter.
			// The MCP SDK resolves it to a temp file path string.
			// But our worker expects "image_path".
			//
			// So if "image" is present, map it to ImagePath.
			if args.Image != "" && args.ImagePath == "" {
				args.ImagePath = args.Image
			}
			elog.Debug("Received args","Image",args.Image,"ImagePath",args.ImagePath,"Query",args.Query,"CollectionID",args.CollectionID)
			return SearchImagesWorker(ctx, req, args, cfg)
		})
}
